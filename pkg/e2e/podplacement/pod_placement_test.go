package podplacement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/multiarch-manager-operator/pkg/e2e"
	"github.com/openshift/multiarch-manager-operator/pkg/testing/framework"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"
)

const helloOpenshiftPublicMultiarchImage = "quay.io/openshifttest/hello-openshift:1.2.0"

var _ = Describe("The Pod Placement Operand", func() {
	Context("When a deployment is deployed with a single container and a public image", func() {
		It("should set the node affinity", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			d := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: ns.Name,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: utils.NewPtr(int32(1)),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: helloOpenshiftPublicMultiarchImage,
								},
							},
						},
					},
				},
			}
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			r, err := labels.NewRequirement("app", "in", []string{"test"})
			labelSelector := labels.NewSelector().Add(*r)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func(g Gomega) {
				pods := &corev1.PodList{}
				err := client.List(ctx, pods, &runtimeclient.ListOptions{
					Namespace:     ns.Name,
					LabelSelector: labelSelector,
				})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(pods.Items).NotTo(BeEmpty())
				g.Expect(pods.Items).To(HaveEach(framework.HaveEquivalentNodeAffinity(
					&corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      utils.ArchLabel,
											Operator: corev1.NodeSelectorOpIn,
											Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le},
										},
									},
								},
							},
						},
					})))
			}, e2e.WaitShort).Should(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
