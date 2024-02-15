package podplacement

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/openshift/multiarch-manager-operator/pkg/utils"

	. "github.com/openshift/multiarch-manager-operator/pkg/testing/framework"
	"github.com/openshift/multiarch-manager-operator/pkg/testing/image/fake/registry"
)

var _ = Describe("Controllers/Podplacement/PodReconciler", func() {
	When("Handling Single-container Pods", func() {
		Context("with different image types", func() {
			DescribeTable("handles correctly", func(imageType string, supportedArchitectures ...string) {
				pod := NewPod().
					WithContainersImages(fmt.Sprintf("%s/%s/%s:latest", registryAddress,
						registry.PublicRepo, registry.ComputeNameByMediaType(imageType))).
					WithGenerateName("test-pod-").
					WithNamespace("test-namespace").
					Build()
				err := k8sClient.Create(ctx, &pod)
				Expect(err).NotTo(HaveOccurred(), "failed to create pod", err)
				// Test the removal of the scheduling gate. However, since the pod is mutated and the reconciler
				// removes the scheduling gate concurrently, we cannot ensure that the scheduling gate is added
				// and that the following Eventually works on a pod with the scheduling gate.
				// Waiting for the pod to be mutated is not enough, as the pod could be mutated and the reconciler
				// could have removed the scheduling gate before our check.
				Eventually(func(g Gomega) {
					// Get pod from the API server
					err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&pod), &pod)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get pod", err)
					g.Expect(pod.Spec.SchedulingGates).NotTo(ContainElement(corev1.PodSchedulingGate{
						Name: schedulingGateName,
					}), "scheduling gate not removed")
					g.Expect(pod.Labels).To(HaveKeyWithValue(utils.SchedulingGateLabel, utils.SchedulingGateLabelValueRemoved),
						"scheduling gate annotation not found")
				}).Should(Succeed(), "failed to remove scheduling gate from pod")
				Eventually(func(g Gomega) {
					g.Expect(pod.Spec.Affinity).NotTo(BeNil(), "pod affinity is nil")
					g.Expect(pod.Spec.Affinity.NodeAffinity).NotTo(BeNil(),
						"pod nodeAffinity is nil")
					g.Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).NotTo(BeNil(),
						"requiredDuringSchedulingIgnoredDuringExecution is nil")
					g.Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).
						NotTo(BeEmpty(), "node selector terms is empty")
					g.Expect(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).
						Should(WithTransform(utils.SortMatchExpressions, ConsistOf(
							utils.SortMatchExpressions([]corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Operator: corev1.NodeSelectorOpIn,
											Values:   supportedArchitectures,
											Key:      utils.ArchLabel,
										},
									},
								},
							}))), "unexpected node selector terms")
				}).Should(Succeed(), "failed to set node affinity to pod")
			},
				Entry("OCI Index Images", imgspecv1.MediaTypeImageIndex, utils.ArchitectureAmd64, utils.ArchitectureArm64),
				Entry("Docker images", imgspecv1.MediaTypeImageManifest, utils.ArchitecturePpc64le),
			)
		},
		)

		Context("with different pull secrets", func() {
			It("handles images with global pull secrets correctly", func() {
				// TODO: Test logic for handling a Pod with one container and image using global pull secret
			})

			It("handles images with local pull secrets correctly", func() {
				// TODO: Test logic for handling a Pod with one container and image using local pull secret
			})
		})
	})

	When("Handling Multi-container Pods", func() {
		Context("with different image types and different auth credentials sources", func() {
			It("handles node affinity as the intersection of the compatible architectures of each multi-arch image", func() {
				// TODO: Test logic for handling a Pod with multiple multi-arch image-based containers
			})

			It("handles node affinity of multi-arch images and single-arch image setting the only one possible", func() {
				// TODO: Test logic for handling a Pod with multiple multi-arch image-based containers
			})

		})
	})
})
