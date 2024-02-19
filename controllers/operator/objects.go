package operator

import (
	"fmt"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/openshift/multiarch-manager-operator/apis/multiarch/v1alpha1"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"
)

func buildMutatingWebhookConfiguration(podPlacementConfig *v1alpha1.PodPlacementConfig) *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: podMutatingWebhookConfigurationName,
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: PodPlacementWebhookName,
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/inject-cabundle": "true",
			},
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      PodPlacementWebhookName,
						Namespace: utils.Namespace(),
						Path:      utils.NewPtr("/add-pod-scheduling-gate"),
					},
				},
				NamespaceSelector: podPlacementConfig.Spec.NamespaceSelector,
				FailurePolicy:     utils.NewPtr(admissionv1.Ignore),
				SideEffects:       utils.NewPtr(admissionv1.SideEffectClassNone),
				Name:              podMutatingWebhookName,
				Rules: []admissionv1.RuleWithOperations{
					{
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
						},
						Rule: admissionv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
			},
		},
	}
}

func buildService(name string, controllerName string, port int32, targetPort intstr.IntOrString) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: controllerName,
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       port,
					TargetPort: targetPort,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: controllerName,
			},
		},
	}
}

func buildDeployment(podPlacementConfig *v1alpha1.PodPlacementConfig,
	name string, replicas int32, serviceAccountName string, args ...string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: utils.Namespace(),
			Labels: map[string]string{
				utils.OperandLabelKey:   operandName,
				utils.ControllerNameKey: name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.NewPtr(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					utils.OperandLabelKey:   operandName,
					utils.ControllerNameKey: name,
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       utils.NewPtr(intstr.FromString("25%")),
					MaxUnavailable: utils.NewPtr(intstr.FromString("25%")),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						utils.OperandLabelKey:   operandName,
						utils.ControllerNameKey: name,
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      utils.ArchLabel,
												Operator: corev1.NodeSelectorOpIn,
												Values: []string{
													utils.ArchitectureAmd64,
													utils.ArchitectureArm64,
													utils.ArchitectureS390x,
													utils.ArchitecturePpc64le,
												},
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           utils.Image(),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: append([]string{
								"--health-probe-bind-address=:8081",
								"--metrics-bind-address=127.0.0.1:8080",
								fmt.Sprintf("-zap-log-level=%d",
									podPlacementConfig.Spec.LogVerbosity.ToZapLevelInt()),
							}, args...),
							Command: []string{
								"/manager",
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt32(8081),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 15,
								TimeoutSeconds:      1,
								PeriodSeconds:       20,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromInt32(8081),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: utils.NewPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
								Privileged:   utils.NewPtr(false),
								RunAsNonRoot: utils.NewPtr(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "webhook-server-cert",
									MountPath: "/var/run/manager/tls",
									ReadOnly:  true,
								},
								{
									Name:      "ca-projected-volume",
									MountPath: "/etc/ssl/certs",
									ReadOnly:  true,
								},
							},
						}, {
							Name:            "kube-rbac-proxy",
							Image:           "gcr.io/kubebuilder/kube-rbac-proxy:v0.13.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--secure-listen-address=0.0.0.0:8443",
								"--upstream=http://127.0.0.1:8080/",
								"--logtostderr=true",
								"--v=0",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8443,
									Protocol:      corev1.ProtocolTCP,
									Name:          "https",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
					ServiceAccountName: serviceAccountName,
					Volumes: []corev1.Volume{
						{
							Name: "webhook-server-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  name,
									DefaultMode: utils.NewPtr(int32(420)),
								},
							},
						},
						{
							Name: "ca-projected-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									DefaultMode: utils.NewPtr(int32(420)),
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "openshift-service-ca.crt",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "service-ca.crt",
														Path: "openshift-ca.crt",
													},
												},
												Optional: utils.NewPtr(true), // Account for the case where the ConfigMap does not exist (non openshift clusters)
											},
										},
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "kube-root-ca.crt",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "ca.crt",
														Path: "kube-root-ca.crt",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
