/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"fmt"

	"github.com/openshift/multiarch-manager-operator/apis/multiarch/v1alpha1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/multiarch-manager-operator/pkg/utils"
)

func validateReconcile() {
	Eventually(func(g Gomega) {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementControllerName,
				Namespace: utils.Namespace(),
			},
		}), deployment)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementControllerName, err)
	}).Should(Succeed(), "the deployment "+PodPlacementControllerName+" should be created")
	Eventually(func(g Gomega) {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementWebhookName,
				Namespace: utils.Namespace(),
			},
		}), deployment)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementControllerName, err)
	}).Should(Succeed(), "the deployment "+PodPlacementWebhookName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementWebhookName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get service "+PodPlacementWebhookName, err)
	}).Should(Succeed(), "the service "+PodPlacementWebhookName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podPlacementControllerMetricsServiceName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get service "+podPlacementControllerMetricsServiceName, err)
	}).Should(Succeed(), "the service "+podPlacementControllerMetricsServiceName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podPlacementWebhookMetricsServiceName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get service "+podPlacementWebhookMetricsServiceName, err)
	}).Should(Succeed(), "the service "+podPlacementWebhookMetricsServiceName+" should be created")
	Eventually(func(g Gomega) {
		mutatingWebhookConf := &admissionv1.MutatingWebhookConfiguration{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&admissionv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: podMutatingWebhookConfigurationName,
			},
		}), mutatingWebhookConf)
		g.Expect(err).NotTo(HaveOccurred(), "failed to get mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
	}).Should(Succeed(), "the mutating webhook configuration "+podMutatingWebhookConfigurationName+" should be created")
}

func validateDeletion() {
	Eventually(func(g Gomega) {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementControllerName,
				Namespace: utils.Namespace(),
			},
		}), deployment)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the deployment "+PodPlacementControllerName, err)
	}).Should(Succeed(), "the deployment "+PodPlacementControllerName+" should be deleted")
	Eventually(func(g Gomega) {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementWebhookName,
				Namespace: utils.Namespace(),
			},
		}), deployment)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the deployment "+PodPlacementWebhookName, err)
	}).Should(Succeed(), "the deployment "+PodPlacementWebhookName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      PodPlacementWebhookName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the service "+PodPlacementWebhookName, err)
	}).Should(Succeed(), "the service "+PodPlacementWebhookName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podPlacementControllerMetricsServiceName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the service "+podPlacementControllerMetricsServiceName, err)
	}).Should(Succeed(), "the service "+podPlacementControllerMetricsServiceName+" should be created")
	Eventually(func(g Gomega) {
		service := &corev1.Service{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podPlacementWebhookMetricsServiceName,
				Namespace: utils.Namespace(),
			},
		}), service)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the service "+podPlacementWebhookMetricsServiceName, err)
	}).Should(Succeed(), "the service "+podPlacementWebhookMetricsServiceName+" should be created")
	Eventually(func(g Gomega) {
		mutatingWebhookConf := &admissionv1.MutatingWebhookConfiguration{}
		err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&admissionv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: podMutatingWebhookConfigurationName,
			},
		}), mutatingWebhookConf)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.IsNotFound(err)).To(BeTrue(), "still getting the mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
	}).Should(Succeed(), "the mutating webhook configuration "+podMutatingWebhookConfigurationName+" should be created")
}

var _ = Describe("Controllers/PodPlacementConfig/PodPlacementConfigReconciler", func() {
	When("The PodPlacementConfig", func() {
		Context("is handling the lifecycle of the operand", func() {
			BeforeEach(func() {
				err := k8sClient.Create(ctx, newPodPlacementConfig().WithName(v1alpha1.SingletonResourceObjectName).Build())
				Expect(err).NotTo(HaveOccurred(), "failed to create PodPlacementConfig", err)
				validateReconcile()
			})
			AfterEach(func() {
				err := k8sClient.Delete(ctx, newPodPlacementConfig().WithName(v1alpha1.SingletonResourceObjectName).Build())
				Expect(err).NotTo(HaveOccurred(), "failed to delete PodPlacementConfig", err)
				validateDeletion()
			})
			It("should refuse to create a PodPlacementConfig with an invalid name", func() {
				ppc := newPodPlacementConfig().WithName("invalid-name").Build()
				err := k8sClient.Create(ctx, ppc)
				Expect(err).NotTo(HaveOccurred(), "failed to create PodPlacementConfig", err)
				Eventually(func(g Gomega) {
					ppc := &v1alpha1.PodPlacementConfig{}
					err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&v1alpha1.PodPlacementConfig{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "invalid-name",
							Namespace: utils.Namespace(),
						},
					}), ppc)
					g.Expect(err).To(HaveOccurred(), "An error should have been detected")
					g.Expect(crclient.IgnoreNotFound(err)).NotTo(HaveOccurred(),
						"The error should not be different than a NotFound error")
				}).Should(Succeed())
			})
			It("should reconcile the deployment pod-placement-controller", func() {
				// get the deployment
				d := appsv1.Deployment{}
				err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      PodPlacementControllerName,
						Namespace: utils.Namespace(),
					},
				}), &d)
				Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementControllerName, err)
				// change the deployment's replicas
				d.Spec.Replicas = utils.NewPtr(int32(3))
				err = k8sClient.Update(ctx, &d)
				Expect(err).NotTo(HaveOccurred(), "failed to update deployment "+PodPlacementControllerName, err)
				Eventually(func(g Gomega) {
					d := appsv1.Deployment{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementControllerName,
							Namespace: utils.Namespace(),
						},
					}), &d)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementControllerName, err)
					g.Expect(d.Spec.Replicas).To(Equal(utils.NewPtr(int32(2))), "the deployment's replicas should be 2")
				}).Should(Succeed(), "the deployment's replicas should be 2")
			})
			It("should reconcile the deployment pod-placement-webhook when deleted", func() {
				err := k8sClient.Delete(ctx, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      PodPlacementWebhookName,
						Namespace: utils.Namespace(),
					},
				}, crclient.PropagationPolicy(metav1.DeletePropagationBackground))
				Expect(err).NotTo(HaveOccurred(), "failed to delete deployment "+PodPlacementWebhookName, err)
				Eventually(func(g Gomega) {
					d := appsv1.Deployment{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementWebhookName,
							Namespace: utils.Namespace(),
						},
					}), &d)
					g.Expect(err).NotTo(HaveOccurred(), "Unable to get deployment "+PodPlacementWebhookName, err)
				}).Should(Succeed(), "the deployment "+PodPlacementWebhookName+" should be recreated")
			})
			It("should reconcile a service if deleted", func() {
				err := k8sClient.Delete(ctx, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      PodPlacementWebhookName,
						Namespace: utils.Namespace(),
					},
				}, crclient.PropagationPolicy(metav1.DeletePropagationBackground))
				Expect(err).NotTo(HaveOccurred(), "failed to delete service "+PodPlacementWebhookName, err)
				Eventually(func(g Gomega) {
					s := &corev1.Service{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementWebhookName,
							Namespace: utils.Namespace(),
						},
					}), s)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get service "+PodPlacementWebhookName, err)
				}).Should(Succeed(), "the service "+PodPlacementWebhookName+" should be recreated")
			})
			It("should reconcile a service if changed", func() {
				s := &corev1.Service{}
				err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      PodPlacementWebhookName,
						Namespace: utils.Namespace(),
					},
				}), s)
				Expect(err).NotTo(HaveOccurred(), "failed to get service "+PodPlacementWebhookName, err)
				By("changing the service's port")
				// change the service's port
				s.Spec.Ports[0].Port = 8080
				err = k8sClient.Update(ctx, s)
				Expect(err).NotTo(HaveOccurred(), "failed to update service "+PodPlacementWebhookName, err)
				By("waiting for the service's port to be reconciled")
				Eventually(func(g Gomega) {
					s := &corev1.Service{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementWebhookName,
							Namespace: utils.Namespace(),
						},
					}), s)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get service "+PodPlacementWebhookName, err)
					g.Expect(s.Spec.Ports[0].Port).To(Equal(int32(443)), "the service's port should be 443")
				}).Should(Succeed(), "the service's port never reconciled to 443")
			})
			It("should reconcile a mutating webhook configuration if changed", func() {
				mwc := &admissionv1.MutatingWebhookConfiguration{}
				err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&admissionv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: podMutatingWebhookConfigurationName,
					},
				}), mwc)
				Expect(err).NotTo(HaveOccurred(), "failed to get mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
				// change the mutating webhook configuration's failure policy
				mwc.Webhooks[0].FailurePolicy = utils.NewPtr(admissionv1.Fail)
				err = k8sClient.Update(ctx, mwc)
				Expect(err).NotTo(HaveOccurred(), "failed to update mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
				Eventually(func(g Gomega) {
					mwc := &admissionv1.MutatingWebhookConfiguration{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&admissionv1.MutatingWebhookConfiguration{
						ObjectMeta: metav1.ObjectMeta{
							Name: podMutatingWebhookConfigurationName,
						},
					}), mwc)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
					g.Expect(mwc.Webhooks[0].FailurePolicy).To(Equal(utils.NewPtr(admissionv1.Ignore)), "the mutating webhook configuration's failure policy should be Ignore")
				}).Should(Succeed(), "the mutating webhook configuration's failure policy never reconciled to Ignore")
			})
			It("should sync the deployments' logLevel arguments", func() {
				// get the podplacementconfig
				ppc2 := &v1alpha1.PodPlacementConfig{}
				err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&v1alpha1.PodPlacementConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      v1alpha1.SingletonResourceObjectName,
						Namespace: utils.Namespace(),
					},
				}), ppc2)
				Expect(err).NotTo(HaveOccurred(), "failed to get PodPlacementConfig", err)
				// change the podplacementconfig's logLevel
				ppc2.Spec.LogVerbosity = v1alpha1.LogVerbosityLevelTraceAll
				err = k8sClient.Update(ctx, ppc2)
				Expect(err).NotTo(HaveOccurred(), "failed to update PodPlacementConfig", err)
				Eventually(func(g Gomega) {
					d := appsv1.Deployment{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementControllerName,
							Namespace: utils.Namespace(),
						},
					}), &d)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementControllerName, err)
					g.Expect(d.Spec.Template.Spec.Containers[0].Args).To(ContainElement(
						fmt.Sprintf("-zap-log-level=%d", v1alpha1.LogVerbosityLevelTraceAll.ToZapLevelInt())))
				}).Should(Succeed(), "the deployment "+PodPlacementControllerName+" should be updated")
				Eventually(func(g Gomega) {
					d := appsv1.Deployment{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodPlacementWebhookName,
							Namespace: utils.Namespace(),
						},
					}), &d)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get deployment "+PodPlacementWebhookName, err)
					g.Expect(d.Spec.Template.Spec.Containers[0].Args).To(ContainElement(
						fmt.Sprintf("-zap-log-level=%d", v1alpha1.LogVerbosityLevelTraceAll.ToZapLevelInt())))
				}).Should(Succeed(), "the deployment "+PodPlacementWebhookName+" should be updated")
			})
			It("Should sync the namespace selector", func() {
				// get the podplacementconfig
				ppc := &v1alpha1.PodPlacementConfig{}
				err := k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&v1alpha1.PodPlacementConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      v1alpha1.SingletonResourceObjectName,
						Namespace: utils.Namespace(),
					},
				}), ppc)
				Expect(err).NotTo(HaveOccurred(), "failed to get PodPlacementConfig", err)
				// change the podplacementconfig's namespace selector
				ppc.Spec.NamespaceSelector = &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"foo": "bar",
					},
				}
				err = k8sClient.Update(ctx, ppc)
				Expect(err).NotTo(HaveOccurred(), "failed to update PodPlacementConfig", err)
				Eventually(func(g Gomega) {
					mw := &admissionv1.MutatingWebhookConfiguration{}
					err = k8sClient.Get(ctx, crclient.ObjectKeyFromObject(&admissionv1.MutatingWebhookConfiguration{
						ObjectMeta: metav1.ObjectMeta{
							Name: podMutatingWebhookConfigurationName,
						},
					}), mw)
					g.Expect(err).NotTo(HaveOccurred(), "failed to get mutating webhook configuration "+podMutatingWebhookConfigurationName, err)
					g.Expect(mw.Webhooks[0].NamespaceSelector).To(Equal(ppc.Spec.NamespaceSelector))
				}).Should(Succeed(), "the deployment "+PodPlacementControllerName+" should be updated")
			})
		})
	})
})

type podPlacementConfigFactory struct {
	*v1alpha1.PodPlacementConfig
}

func newPodPlacementConfig() *podPlacementConfigFactory {
	return &podPlacementConfigFactory{
		PodPlacementConfig: &v1alpha1.PodPlacementConfig{},
	}
}

func (p *podPlacementConfigFactory) WithName(name string) *podPlacementConfigFactory {
	p.Name = name
	return p
}

func (p *podPlacementConfigFactory) WithNamespaceSelector(labelSelector *metav1.LabelSelector) *podPlacementConfigFactory {
	p.Spec.NamespaceSelector = labelSelector
	return p
}

func (p *podPlacementConfigFactory) WithLogVerbosity(logVerbosity v1alpha1.LogVerbosityLevel) *podPlacementConfigFactory {
	p.Spec.LogVerbosity = logVerbosity
	return p
}

func (p *podPlacementConfigFactory) Build() *v1alpha1.PodPlacementConfig {
	return p.PodPlacementConfig
}
