package cmd

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var scheme = runtime.NewScheme()

var testThreshold time.Duration
var cancelCluster context.CancelFunc

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(1 * time.Minute)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "Test zombie-detector", Label("envtest", "test zombie-detector"))
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.Level(-10))))

	var err error

	testThreshold, err = time.ParseDuration("5s")
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	var ctx context.Context
	ctx, cancelCluster = context.WithCancel(context.Background())

	testNamespace := corev1.Namespace{}
	testNamespace.Name = "test"

	testPod := corev1.Pod{}
	testPod.Name = "test-pod"
	testPod.Namespace = "test"
	testPod.Finalizers = []string{"kubernetes"}
	testPod.Spec.Containers = []corev1.Container{{Name: "c1", Image: "nginx"}}

	testConfigMap := v1.ConfigMap{}
	testConfigMap.Name = "test-configmap"
	testConfigMap.Namespace = "test"
	testConfigMap.Finalizers = []string{"kubernetes"}

	testService := v1.Service{}
	testService.Name = "test-service"
	testService.Namespace = "test"
	testService.Finalizers = []string{"kubernetes"}

	testService.Spec = v1.ServiceSpec{
		Type: v1.ServiceTypeClusterIP,
		Ports: []v1.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.IntOrString{IntVal: 80},
			},
		},
	}

	err = k8sClient.Create(ctx, &testNamespace)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(ctx, &testPod)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(ctx, &testConfigMap)
	Expect(err).NotTo(HaveOccurred())
	err = k8sClient.Create(ctx, &testService)
	Expect(err).NotTo(HaveOccurred())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancelCluster()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
