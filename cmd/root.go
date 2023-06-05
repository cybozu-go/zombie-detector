package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	rootCmd = &cobra.Command{
		Use:     "zombie-detector",
		Short:   "zombie-detector detects longly undeleted kubernetes resources",
		RunE:    rootMain,
		Version: "0.0.1",
	}
)

var thresholdFlag time.Duration
var pushgatewayEndpointFlag string

func init() {
	rootCmd.Flags().DurationVar(&thresholdFlag, "threshold", time.Duration(24*time.Hour), "threshold of detection")
	rootCmd.Flags().StringVar(&pushgatewayEndpointFlag, "pushgateway", "", "URL of Pushgateway's endpoint")

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func getAllResources(ctx context.Context, config *rest.Config) ([]unstructured.Unstructured, error) {
	o, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	serverResources, err := o.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	resources := make([]unstructured.Unstructured, 0)
	for _, resList := range serverResources {
		gv, err := schema.ParseGroupVersion(resList.GroupVersion)
		if err != nil {
			gv = schema.GroupVersion{}
		}
		for _, resource := range resList.APIResources {
			groupResourceDef := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			listResponse, err := dynamicClient.Resource(groupResourceDef).Namespace(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
			statusErr := &apierrors.StatusError{}
			if err != nil && !errors.As(err, &statusErr) {
				return nil, err
			}
			if statusErr.ErrStatus.Reason == metav1.StatusReasonNotFound || statusErr.ErrStatus.Reason == metav1.StatusReasonMethodNotAllowed {
				continue
			}
			resources = append(resources, listResponse.Items...)
		}
	}
	return resources, nil
}

func printAllResources(resources []unstructured.Unstructured) {
	for _, res := range resources {
		apiVersion := res.GetAPIVersion()
		kind := res.GetKind()
		name := res.GetName()
		namespace := res.GetNamespace()
		deletionTimestamp := res.GetDeletionTimestamp()
		fmt.Printf("%s, %s, %s,%s %v\n", apiVersion, kind, name, namespace, deletionTimestamp)
	}
}

func detectZombieResource(resource unstructured.Unstructured, threshold time.Duration) bool {
	deletionTimestamp := resource.GetDeletionTimestamp()
	if deletionTimestamp == nil {
		return false
	}
	if time.Since(deletionTimestamp.Time) > threshold {
		return true
	}
	return false
}

func detectZombieResources(resources []unstructured.Unstructured, threshold time.Duration) []unstructured.Unstructured {
	zombieResources := make([]unstructured.Unstructured, 0)
	for _, res := range resources {
		isZombie := detectZombieResource(res, threshold)
		if isZombie {
			zombieResources = append(zombieResources, res)
		}
	}
	return zombieResources
}

func postZombieResourcesMetrics(zombieResources []unstructured.Unstructured, endpoint string) error {
	err := push.New(endpoint, "zombie-detector").Delete()
	if err != nil {
		return err
	}
	if len(zombieResources) == 0 {
		return nil
	}
	gauges := make([]prometheus.Gauge, 0)
	for _, res := range zombieResources {
		apiVersion := res.GetAPIVersion()
		kind := res.GetKind()
		name := res.GetName()
		namespace := res.GetNamespace()
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zombie_duration_hours",
			Help: "zombie detector zombie duration",
			ConstLabels: map[string]string{
				"apiVersion": apiVersion,
				"kind":       kind,
				"name":       name,
				"namespace":  namespace,
				"updated_at": time.Now().Format(time.RFC3339),
			},
		})
		deletionTimestamp := res.GetDeletionTimestamp()
		gauge.Set(time.Since(deletionTimestamp.Time).Seconds())
		gauges = append(gauges, gauge)
	}
	registry := prometheus.NewRegistry()
	for _, g := range gauges {
		registry.MustRegister(g)
	}
	err = push.New(endpoint, "zombie-detector").Gatherer(registry).Add()
	if err != nil {
		return err
	}
	return nil
}

func rootMain(cmd *cobra.Command, args []string) error {
	config, err := config.GetConfig()
	if err != nil {
		return err
	}
	ctx := context.Background()
	allResources, err := getAllResources(ctx, config)
	if err != nil {
		return err
	}
	zombieResources := detectZombieResources(allResources, thresholdFlag)
	err = postZombieResourcesMetrics(zombieResources, pushgatewayEndpointFlag)
	if err != nil {
		return err
	}

	return nil
}
