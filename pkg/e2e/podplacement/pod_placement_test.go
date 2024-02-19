package podplacement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
			verifyPodNodeAffinity(ns, "app", "test", utils.ArchitectureAmd64,
				utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le)
		})
		It("should set the node affinity on privileged deployments", func() {
			var err error
			ns := framework.NewEphemeralNamespace()
			ns.Labels = map[string]string{
				"pod-security.kubernetes.io/audit":               "privileged",
				"pod-security.kubernetes.io/warn":                "privileged",
				"pod-security.kubernetes.io/enforce":             "privileged",
				"security.openshift.io/scc.podSecurityLabelSync": "false",
			}
			err = client.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, ns)
			// Since the CRB to enable using SCCs is cluster-scoped, build its name based on the NS to
			// ensure concurrency safety with other possible test cases
			ephemeralCRBName := framework.NormalizeNameString("priv-scc-" + ns.Name)

			crb := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: ephemeralCRBName,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     "system:openshift:scc:privileged",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      rbacv1.ServiceAccountKind,
						Name:      "default",
						Namespace: ns.Name,
					},
				},
			}
			err = client.Create(ctx, crb)
			Expect(err).NotTo(HaveOccurred())
			//nolint:errcheck
			defer client.Delete(ctx, crb)

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
									SecurityContext: &corev1.SecurityContext{
										Privileged: utils.NewPtr(true),
										RunAsGroup: utils.NewPtr(int64(0)),
										RunAsUser:  utils.NewPtr(int64(0)),
										SeccompProfile: &corev1.SeccompProfile{
											Type: corev1.SeccompProfileTypeUnconfined,
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "test-hostpath",
											MountPath: "/mnt/hostpath",
										},
									},
								},
							},
							ServiceAccountName: "default",
							Volumes: []corev1.Volume{
								{
									Name: "test-hostpath",
									VolumeSource: corev1.VolumeSource{
										HostPath: &corev1.HostPathVolumeSource{
											Path: "/var/lib/kubelet/config.json",
											Type: utils.NewPtr(corev1.HostPathFile),
										},
									},
								},
							},
						},
					},
				},
			}
			err = client.Create(ctx, &d)
			Expect(err).NotTo(HaveOccurred())
			// TODO: verify the pod has some scc label
			r, err := labels.NewRequirement("app", "in", []string{"test"})
			Expect(err).NotTo(HaveOccurred())
			labelSelector := labels.NewSelector().Add(*r)
			Eventually(func(g Gomega) {
				pods := &corev1.PodList{}
				err := client.List(ctx, pods, &runtimeclient.ListOptions{
					Namespace:     ns.Name,
					LabelSelector: labelSelector,
				})
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(pods.Items).NotTo(BeEmpty())
				pod := pods.Items[0]
				g.Expect(pod.Annotations).NotTo(BeEmpty())
				g.Expect(pod.Annotations).To(HaveKeyWithValue("openshift.io/scc", "privileged"))
			})
			verifyPodNodeAffinity(ns, "app", "test", utils.ArchitectureAmd64,
				utils.ArchitectureArm64, utils.ArchitectureS390x, utils.ArchitecturePpc64le)
		})
	})
})

func verifyPodNodeAffinity(ns *corev1.Namespace, labelKey string, labelInValue string, supportedArch ...string) {
	r, err := labels.NewRequirement(labelKey, "in", []string{labelInValue})
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
									Values:   supportedArch,
								},
							},
						},
					},
				},
			})))
	}, e2e.WaitShort).Should(Succeed())
}
