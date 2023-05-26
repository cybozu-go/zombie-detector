package e2e

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

func TestE2e(t *testing.T) {
	if !runE2E {
		t.Skip("no RUN_E2E environment variable")
	}
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(100 * time.Millisecond)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	By("creating resources from manifests")
	_, err := kubectl(nil, "apply", "-f", "./manifests/pod.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubectl(nil, "apply", "-f", "./manifests/deployment.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubectl(nil, "apply", "-f", "./manifests/configmap.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubectl(nil, "apply", "-f", "./manifests/observer_pod.yaml")
	Expect(err).NotTo(HaveOccurred())

	By("applying cronjob manifest")
	_, err = kubectl(nil, "apply", "-f", "../config/cronjob/cronjob.yaml")
	Expect(err).NotTo(HaveOccurred())

	res, err := kubectl(nil, "get", "cronjob", "zombie-detector-cronjob", "-n", "zombie-detector", "-o", "json")
	Expect(err).NotTo(HaveOccurred())
	cron := batchv1.CronJob{}
	err = json.Unmarshal(res, &cron)
	Expect(err).NotTo(HaveOccurred())
	Expect(cron.Name).To(Equal("zombie-detector-cronjob"))

	By("waiting for pushgateway to be started")
	Eventually(func() error {
		res, err := kubectl(nil, "get", "deployment", "pushgateway", "-n", "monitoring", "-o", "json")
		deploy := appsv1.Deployment{}
		err = json.Unmarshal(res, &deploy)
		Expect(err).NotTo(HaveOccurred())
		if deploy.Status.ReadyReplicas == 0 {
			return fmt.Errorf("pushgateway is not ready yet")
		}
		return nil
	}).Should(Succeed())

})

var _ = AfterSuite(func() {
	By("deleting test resources")
	_, err := kubectl(nil, "patch", "deployment", "test-deployment", "--patch-file", "./manifests/patch.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubectl(nil, "patch", "pod", "test-pod", "--patch-file", "./manifests/patch.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = kubectl(nil, "patch", "configmap", "test-configmap", "--patch-file", "./manifests/patch.yaml")
	Expect(err).NotTo(HaveOccurred())
})