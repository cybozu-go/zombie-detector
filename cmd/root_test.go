package cmd

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDetectZombieResource(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name           string
		resource       runtime.Object
		thresholdHours string
		want           bool
	}{
		{
			name: "No problem Pod",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: nil,
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie below the threshold Pod",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-22 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie over the threshold Pod",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-26 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           true,
		},
		{
			name: "Zombie over the threshold Pod with finalizer",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-26 * time.Hour)},
					Finalizers:        []string{"kubernetes"},
				},
			},
			thresholdHours: "24h",
			want:           true,
		},
		{
			name: "Zombie over the threshold(changed) Pod",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
					Finalizers:        []string{"kubernetes"},
				},
			},
			thresholdHours: "24h",
			want:           true,
		},
		{
			name: "Zombie threshold boundary Pod",
			resource: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-13 * time.Hour)},
					Finalizers:        []string{"kubernetes"},
				},
			},
			thresholdHours: "12h",
			want:           true,
		},
		{
			name: "No problem Deployment",
			resource: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: nil,
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie below the threshold Deployment",
			resource: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deoloyment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-22 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie over the threshold Deployment",
			resource: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deoloyment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-26 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           true,
		},
		{
			name: "No problem ConfigMap",
			resource: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: nil,
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie below the threshold ConfigMap",
			resource: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-22 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           false,
		},
		{
			name: "Zombie over the threshold Deployment",
			resource: &v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-resource",
					Namespace:         "test",
					DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-26 * time.Hour)},
					Finalizers:        nil,
				},
			},
			thresholdHours: "24h",
			want:           true,
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.resource)
			require.NoError(t, err)
			unstructuredObj := unstructured.Unstructured{
				Object: obj,
			}
			threshold, err := time.ParseDuration(tt.thresholdHours)
			require.NoError(t, err)

			r := detectZombieResource(unstructuredObj, threshold)
			assert.Equal(t, tt.want, r)
		})
	}
}

var _ = Describe("Test zombie-detector", func() {
	ctx := context.Background()
	It("should not detect anything", func() {
		allResources, err := getAllResources(ctx, cfg)
		Expect(err).NotTo(HaveOccurred())
		zombieResources := detectZombieResources(allResources, testThreshold)
		Expect(zombieResources).To(BeEmpty())
	})

	It("should detect zombie resources", func() {
		By("deleting test pod")
		testPod := corev1.Pod{}
		testPod.Name = "test-pod"
		testPod.Namespace = "test"
		err := k8sClient.Delete(ctx, &testPod)
		Expect(err).NotTo(HaveOccurred())

		By("deleting test configMap")
		testConfigMap := v1.ConfigMap{}
		testConfigMap.Name = "test-configmap"
		testConfigMap.Namespace = "test"
		err = k8sClient.Delete(ctx, &testConfigMap)
		Expect(err).NotTo(HaveOccurred())

		By("checking deletionTimestamp")
		checkZombie := func(resource client.Object) error {
			if resource.GetDeletionTimestamp() == nil {
				return fmt.Errorf("resource must have deletionTimestamp")
			}
			if time.Since(resource.GetDeletionTimestamp().Time) < testThreshold {
				return fmt.Errorf("it should wait to pass the threshold time")
			}
			return nil
		}
		resources := []types.NamespacedName{
			{
				Name:      "test-pod",
				Namespace: "test",
			},
			{
				Name:      "test-configmap",
				Namespace: "test",
			},
		}
		Eventually(func() error {
			testPod := corev1.Pod{}
			testConfigMap := v1.ConfigMap{}

			if err := k8sClient.Get(ctx, resources[0], &testPod); err != nil {
				return err
			}
			if err := k8sClient.Get(ctx, resources[1], &testConfigMap); err != nil {
				return err
			}
			if err := checkZombie(&testPod); err != nil {
				return err
			}
			if err := checkZombie(&testConfigMap); err != nil {
				return err
			}
			return nil
		}).Should(Succeed())

		By("detecting zombie pod")
		allResources, err := getAllResources(ctx, cfg)
		Expect(err).NotTo(HaveOccurred())
		zombieResources := detectZombieResources(allResources, testThreshold)
		Expect(len(zombieResources)).To(Equal(2))

		By("checking finalizers and deletionTimestamp exist")
		for _, res := range zombieResources {
			finalizers := res.GetFinalizers()
			Expect(finalizers).NotTo(BeEmpty())
			deletionTimestamp := res.GetDeletionTimestamp()
			Expect(deletionTimestamp).NotTo(BeNil())
		}
	})
})
