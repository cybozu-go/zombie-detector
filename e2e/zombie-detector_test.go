package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ = Describe("zombie-detector e2e test", func() {
	AfterEach(func() {
		_, err := kubectl(nil, "delete", "job", "zombie-detector-immediate-job", "-n", "zombie-detector")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not detect anything", func() {
		By("creating job from cronjob")
		_, err := kubectl(nil, "create", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "--from=cronjob/zombie-detector-cronjob")
		Expect(err).NotTo(HaveOccurred())

		By("waiting for job to be completed")
		Eventually(func() error {
			res, err := kubectl(nil, "get", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "-o", "json")
			job := batchv1.Job{}
			err = json.Unmarshal(res, &job)
			if err != nil {
				return err
			}
			if job.Status.Succeeded == 0 {
				return fmt.Errorf("job is not completed yet")
			}
			return nil
		}).Should(Succeed())
		By("checking metrics is empty")
		res, err := getMetricsFromPushgateway()
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Data).To(BeEmpty())
	})

	It("should detect zombie resources", func() {
		By("adding deletionTimestamp to resources")
		_, err := kubectl(nil, "delete", "deployment", "test-deployment", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "delete", "pod", "test-pod", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "delete", "configmap", "test-configmap", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())

		By("waiting for time passing since deletionTimestamp is added")
		Eventually(func() error {
			res, err := kubectl(nil, "get", "pod", "test-pod", "-n", "default", "-o", "json")
			if err != nil {
				return err
			}
			pod := corev1.Pod{}
			err = json.Unmarshal(res, &pod)
			if err != nil {
				return err
			}
			res, err = kubectl(nil, "get", "deployment", "test-deployment", "-n", "default", "-o", "json")
			if err != nil {
				return err
			}
			deploy := appsv1.Deployment{}
			err = json.Unmarshal(res, &deploy)
			if err != nil {
				return err
			}
			res, err = kubectl(nil, "get", "configmap", "test-configmap", "-n", "default", "-o", "json")
			if err != nil {
				return err
			}
			configmap := corev1.ConfigMap{}
			err = json.Unmarshal(res, &configmap)
			if err != nil {
				return err
			}
			if time.Since(pod.GetDeletionTimestamp().Time).Seconds() < 5 || time.Since(deploy.GetDeletionTimestamp().Time).Seconds() < 5 || time.Since(configmap.GetDeletionTimestamp().Time).Seconds() < 5 {
				return fmt.Errorf("at least one resource is not deleted yet")
			}
			return nil
		}).Should(Succeed())

		By("creating job from cronjob")
		_, err = kubectl(nil, "create", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "--from=cronjob/zombie-detector-cronjob")
		Expect(err).NotTo(HaveOccurred())

		By("waiting for job to be completed")
		Eventually(func() error {
			res, err := kubectl(nil, "get", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "-o", "json")
			job := batchv1.Job{}
			err = json.Unmarshal(res, &job)
			if err != nil {
				return err
			}
			if job.Status.Succeeded == 0 {
				return fmt.Errorf("job is not completed yet")
			}
			return nil
		}).Should(Succeed())
		By("checking metrics has 3 entries")
		res, err := getMetricsFromPushgateway()
		Expect(err).NotTo(HaveOccurred())
		index, err := returnZombieDetectorMetricsIndex(*res)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(res.Data[index].ZombieDurationSeconds.Metrics)).To(Equal(3))
	})

	It("should not detect anything again", func() {
		By("deleting test resources by deleting finalizers")
		_, err := kubectl(nil, "patch", "deployment", "test-deployment", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "patch", "pod", "test-pod", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "patch", "configmap", "test-configmap", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())

		By("check resources are completely deleted")
		Eventually(func() error {
			_, err := kubectl(nil, "get", "pod", "test-pod", "-n", "default")
			if err == nil { //kubectl returns error when resource does not exists.
				return fmt.Errorf("pod is not deleted yet")
			}
			_, err = kubectl(nil, "get", "deployment", "test-deployment", "-n", "default")
			if err == nil {
				return fmt.Errorf("pod is not deleted yet")
			}
			_, err = kubectl(nil, "get", "pod", "test-configmap", "-n", "default")
			if err == nil {
				return fmt.Errorf("pod is not deleted yet")
			}
			return nil
		}).Should(Succeed())

		By("creating job from cronjob")
		_, err = kubectl(nil, "create", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "--from=cronjob/zombie-detector-cronjob")
		Expect(err).NotTo(HaveOccurred())

		By("waiting for job to be completed")
		Eventually(func() error {
			res, err := kubectl(nil, "get", "job", "zombie-detector-immediate-job", "-n", "zombie-detector", "-o", "json")
			if err != nil {
				return err
			}
			job := batchv1.Job{}
			err = json.Unmarshal(res, &job)
			if err != nil {
				return err
			}
			if job.Status.Succeeded == 0 {
				return fmt.Errorf("job is not completed yet")
			}
			return nil
		}).Should(Succeed())

		By("checking metrics is empty")
		res, err := getMetricsFromPushgateway()
		Expect(err).NotTo(HaveOccurred())
		Expect(res.Data).To(BeEmpty())
	})

})

func getMetricsFromPushgateway() (*Response, error) {
	res, err := kubectl(nil, "exec", "-n", "monitoring", "observer-pod", "--", "curl", "-sS", "-m", "30", "http://pushgateway.monitoring.svc.cluster.local:9091/api/v1/metrics")
	if err != nil {
		return nil, err
	}
	response := &Response{}
	if err := json.Unmarshal(res, &response); err != nil {
		return nil, err
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("unexpected status: %s", response.Status)
	}
	return response, nil
}

func returnZombieDetectorMetricsIndex(res Response) (int, error) {
	for iter, d := range res.Data {
		if d.Labels.Job == "zombie-detector" {
			return iter, nil
		}
	}
	return -1, fmt.Errorf("zombie-detector metrics not found")
}

type Response struct {
	Status string `json:"status"`
	Data   []struct {
		Labels struct {
			Job string `json:"job"`
		} `json:"labels"`
		ZombieDurationSeconds struct {
			Metrics []struct {
				Labels struct {
					APIVersion string    `json:"apiVersion"`
					Instance   string    `json:"instance"`
					Job        string    `json:"job"`
					Kind       string    `json:"kind"`
					Name       string    `json:"name"`
					Namespace  string    `json:"namespace"`
					UpdatedAt  time.Time `json:"updated_at"`
				} `json:"labels"`
				Value string `json:"value"`
			} `json:"metrics"`
		} `json:"zombie_duration_seconds,omitempty"`
	} `json:"data"`
}
