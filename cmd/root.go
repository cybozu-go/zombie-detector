package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Version: "1.1.0",
	}
)

var thresholdFlag time.Duration
var pushgatewayEndpointFlag string

func init() {
	rootCmd.Flags().DurationVar(&thresholdFlag, "threshold", time.Duration(24*time.Hour), "threshold of detection")
	rootCmd.MarkFlagRequired("threshold")
	rootCmd.Flags().StringVar(&pushgatewayEndpointFlag, "pushgateway", "", "URL of Pushgateway's endpoint. If this flag is not given, the result outputs to stdout")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

type resourceMetadata struct {
	version           string
	kind              string
	name              string
	namespace         string
	deletionTimestamp *metav1.Time
}

var IgnoreResources = []schema.GroupVersionResource{
	{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "pods",
	},
	{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "nodes",
	},
}

func getAllResources(ctx context.Context, config *rest.Config) ([]resourceMetadata, error) {
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
	resources := make([]resourceMetadata, 0)
	for _, resList := range serverResources {
		gv, err := schema.ParseGroupVersion(resList.GroupVersion)
		if err != nil {
			gv = schema.GroupVersion{}
		}
	L:
		for _, resource := range resList.APIResources {
			groupResourceDef := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: resource.Name}
			for _, ir := range IgnoreResources {
				if ir == groupResourceDef {
					fmt.Printf("ignoring %s %s %s\n", groupResourceDef.Group, groupResourceDef.Version, groupResourceDef.Resource)
					continue L
				}
			}
			listResponse, err := dynamicClient.Resource(groupResourceDef).Namespace(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
			statusErr := &apierrors.StatusError{}
			if err != nil && !errors.As(err, &statusErr) {
				return nil, err
			}
			if statusErr.ErrStatus.Reason == metav1.StatusReasonNotFound || statusErr.ErrStatus.Reason == metav1.StatusReasonMethodNotAllowed {
				continue
			}
			for _, item := range listResponse.Items {
				resources = append(resources, resourceMetadata{
					version:           item.GetAPIVersion(),
					kind:              item.GetKind(),
					name:              item.GetName(),
					namespace:         item.GetNamespace(),
					deletionTimestamp: item.GetDeletionTimestamp(),
				})
			}
		}
	}
	return resources, nil
}

func printAllResources(resources []resourceMetadata) {
	data := make([][]string, 0, len(resources))
	for _, res := range resources {
		data = append(data, []string{res.version, res.kind, res.name, res.namespace, res.deletionTimestamp.String()})
	}
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.SeparatorsNone,
				Lines:      tw.LinesNone,
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{Alignment: tw.AlignLeft},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{Alignment: tw.AlignLeft},
			},
		}),
	)
	table.Header("Version", "Kind", "Name", "Namespace", "Timestamp")
	table.Bulk(data)
	table.Render()
}

func detectZombieResource(resource resourceMetadata, threshold time.Duration) bool {
	if resource.deletionTimestamp == nil {
		return false
	}
	if time.Since(resource.deletionTimestamp.Time) > threshold {
		return true
	}
	return false
}

func detectZombieResources(resources []resourceMetadata, threshold time.Duration) []resourceMetadata {
	zombieResources := make([]resourceMetadata, 0)
	for _, res := range resources {
		isZombie := detectZombieResource(res, threshold)
		if isZombie {
			zombieResources = append(zombieResources, res)
		}
	}
	return zombieResources
}

func postZombieResourcesMetrics(zombieResources []resourceMetadata, endpoint string) error {
	err := push.New(endpoint, "zombie-detector").Delete()
	if err != nil {
		return err
	}
	if len(zombieResources) == 0 {
		return nil
	}
	gauges := make([]prometheus.Gauge, 0)
	for _, res := range zombieResources {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "zombie_duration_seconds",
			Help: "zombie detector zombie duration",
			ConstLabels: map[string]string{
				"apiVersion": res.version,
				"kind":       res.kind,
				"name":       res.name,
				"namespace":  res.namespace,
				"updated_at": time.Now().Format(time.RFC3339),
			},
		})
		gauge.Set(time.Since(res.deletionTimestamp.Time).Seconds())
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

	if pushgatewayEndpointFlag == "" {
		printAllResources(zombieResources)
		return nil
	}
	err = postZombieResourcesMetrics(zombieResources, pushgatewayEndpointFlag)
	if err != nil {
		return err
	}

	return nil
}
