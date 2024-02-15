package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"

	"github.com/openshift/multiarch-manager-operator/pkg/testing/framework"
)

func CommonInit() {
	err := framework.RegisterScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

func CommonBeforeSuite() (client runtimeclient.Client, clientset *kubernetes.Clientset,
	ctx context.Context, suiteLog logr.Logger) {
	var err error
	client, err = framework.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	clientset, err = framework.LoadClientset()
	Expect(err).ToNot(HaveOccurred())

	suiteLog = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.Level(-5)))
	logf.SetLogger(suiteLog)

	ctx = context.Background()

	SetDefaultEventuallyPollingInterval(PollingInterval)
	SetDefaultEventuallyTimeout(WaitShort)

	return
}
