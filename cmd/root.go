package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	rootCmd = &cobra.Command{
		Use:   "zombie-detector",
		Short: "zombie-detector detects longly undeleted kubernetes resources",
		RunE:  rootMain,
	}
)

const (
	errorFaildtoList      string = "the server could not find the requested resource"
	errorMethodNotAllowed string = "the server does not allow this method on the requested resource"
)

var inClusterFlag bool
var thresholdFlag string
var pushgatewayEndpointFlag string

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Flags().BoolVar(&inClusterFlag, "incluster", true, "execute in cluster or not")
	rootCmd.Flags().StringVar(&thresholdFlag, "threshold", "24h", "threshold of detection")
	rootCmd.Flags().StringVar(&pushgatewayEndpointFlag, "pushgateway", "", "URL of Pushgateway's endpoint")

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func setupClusterConfig(inCluster bool) (*rest.Config, error) {
	if inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	}
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		return nil, errors.New("Faild to load kubeconfig")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
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
			groupeResourceDef := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			listResponse, err := dynamicClient.Resource(groupeResourceDef).Namespace(apiv1.NamespaceAll).List(ctx, metav1.ListOptions{})
			if err != nil {
				if strings.Contains(err.Error(), errorFaildtoList) || strings.Contains(err.Error(), errorMethodNotAllowed) {
					continue
				}
				return nil, err
			}
			resources = append(resources, listResponse.Items...)
		}
	}
	return resources, nil
}

func printAllResources(resources []unstructured.Unstructured) {
	for _, res := range resources {
		apiVersion, found, err := unstructured.NestedString(res.Object, "apiVersion")
		if err != nil || !found {
			continue
		}
		kind, found, err := unstructured.NestedString(res.Object, "kind")
		if err != nil || !found {
			continue
		}
		name, found, err := unstructured.NestedString(res.Object, "metadata", "name")
		if err != nil || !found {
			continue
		}
		namespace, _, err := unstructured.NestedString(res.Object, "metadata", "namespace")
		if err != nil {
			continue
		}
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
		apiVersion, found, err := unstructured.NestedString(res.Object, "apiVersion")
		if err != nil || !found {
			continue
		}
		kind, found, err := unstructured.NestedString(res.Object, "kind")
		if err != nil || !found {
			continue
		}
		name, found, err := unstructured.NestedString(res.Object, "metadata", "name")
		if err != nil || !found {
			continue
		}
		namespace, _, err := unstructured.NestedString(res.Object, "metadata", "namespace")
		if err != nil {
			continue
		}
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
		gauge.Set(time.Since(deletionTimestamp.Time).Hours())
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

	threshold, err := time.ParseDuration(thresholdFlag)
	if err != nil {
		return err
	}
	config, err := setupClusterConfig(inClusterFlag)
	if err != nil {
		return err
	}
	ctx := context.Background()
	allResources, err := getAllResources(ctx, config)
	if err != nil {
		return err
	}
	printAllResources(allResources)
	fmt.Println("---")
	zombieResources := detectZombieResources(allResources, threshold)
	printAllResources(zombieResources)
	err = postZombieResourcesMetrics(zombieResources, pushgatewayEndpointFlag)
	if err != nil {
		return err
	}

	return nil
}
