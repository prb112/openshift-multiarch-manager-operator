package operator_test

import (
	"context"
	"testing"

	"github.com/openshift/multiarch-manager-operator/pkg/e2e"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	client    runtimeclient.Client
	clientset *kubernetes.Clientset
	ctx       context.Context
	suiteLog  = ctrl.Log.WithName("setup")
)

func init() {
	e2e.CommonInit()
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MMO Suite (Operator E2E)", Label("e2e", "operator"))
}

var _ = BeforeSuite(func() {
	client, clientset, ctx, suiteLog = e2e.CommonBeforeSuite()
})
