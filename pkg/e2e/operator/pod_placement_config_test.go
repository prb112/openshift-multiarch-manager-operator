package operator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/multiarch-manager-operator/apis/multiarch/v1alpha1"
	"github.com/openshift/multiarch-manager-operator/controllers/operator"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"
)

var _ = Describe("The Multiarch Manager Operator", func() {
	Context("When the operator is running and a pod placement config is created", func() {
		It("should deploy the operands", func() {
			err := client.Create(ctx, &v1alpha1.PodPlacementConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			// Gets the deployments and checks if they are running
			Eventually(func(g Gomega) {
				d, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
					metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(d.Status.AvailableReplicas).To(Equal(d.Status.Replicas),
					"at least one pod placement controller replicas is not available yet")
				d, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
					metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(d.Status.AvailableReplicas).To(Equal(d.Status.Replicas),
					"at least one pod placement webhook replicas is not available yet")
			}).Should(Succeed())
		})
		AfterEach(func() {
			err := client.Delete(ctx, &v1alpha1.PodPlacementConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				_, err := clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementControllerName,
					metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
				_, err = clientset.AppsV1().Deployments(utils.Namespace()).Get(ctx, operator.PodPlacementWebhookName,
					metav1.GetOptions{})
				g.Expect(err).To(HaveOccurred())
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}).Should(Succeed())
		})
	})
})
