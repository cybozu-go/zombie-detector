package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
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
	It("should detect zombie resouces", func() {
		By("adding deletionTimestamp to resouces")
		_, err := kubectl(nil, "delete", "deployment", "test-deployment", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "delete", "pod", "test-pod", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "delete", "configmap", "test-configmap", "-n", "default", "--wait=false")
		Expect(err).NotTo(HaveOccurred())

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
		Expect(len(res.Data[index].ZombieDurationHours.Metrics)).To(Equal(3))
	})

	It("should not detect anything again", func() {
		By("deleting test resources")
		_, err := kubectl(nil, "patch", "deployment", "test-deployment", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "patch", "pod", "test-pod", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())
		_, err = kubectl(nil, "patch", "configmap", "test-configmap", "--patch-file", "./manifests/patch.yaml")
		Expect(err).NotTo(HaveOccurred())

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
	if response.Staus != "success" {
		return nil, fmt.Errorf("unexpected status: %s", response.Staus)
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
	Staus string `json:"status"`
	Data  []struct {
		Labels struct {
			Job string `json:"job"`
		} `json:"labels"`
		ZombieDurationHours struct {
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
		} `json:"zombie_duration_hours"`
	} `json:"data"`
}
