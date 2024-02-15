package podplacement_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/multiarch-manager-operator/apis/multiarch/v1alpha1"
	"github.com/openshift/multiarch-manager-operator/controllers/operator"
	"github.com/openshift/multiarch-manager-operator/pkg/e2e"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"
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
	RunSpecs(t, "MMO Suite (PodPlacementOperand E2E)", Label("e2e", "pod-placement-operand"))
}

var _ = BeforeSuite(func() {
	client, clientset, ctx, suiteLog = e2e.CommonBeforeSuite()
	err := client.Create(ctx, &v1alpha1.PodPlacementConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(deploymentsAreRunning).Should(Succeed())
})

var _ = AfterSuite(func() {
	err := client.Delete(ctx, &v1alpha1.PodPlacementConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(deploymentsAreDeleted).Should(Succeed())
})

func deploymentsAreRunning(g Gomega) {
	d, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
		metav1.GetOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.Status.AvailableReplicas).To(Equal(*d.Spec.Replicas),
		"at least one pod placement controller replicas is not available yet")
	d, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
		metav1.GetOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(d.Status.AvailableReplicas).To(Equal(*d.Spec.Replicas),
		"at least one pod placement webhook replicas is not available yet")
}

func deploymentsAreDeleted(g Gomega) {
	_, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
		metav1.GetOptions{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
	_, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
		metav1.GetOptions{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
}
