/*
Copyright 2026 Red Hat, Inc.

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

package podplacement

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

func TestApplyArchitectureNodeAffinity(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		architectures  []string
		expectAffinity bool
	}{
		{
			name: "apply single architecture",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			architectures:  []string{"ppc64le"},
			expectAffinity: true,
		},
		{
			name: "apply multiple architectures",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			architectures:  []string{"amd64", "ppc64le"},
			expectAffinity: true,
		},
		{
			name: "empty architectures list",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			architectures:  []string{},
			expectAffinity: false,
		},
		{
			name: "apply to pod with existing affinity",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{},
					},
				},
			},
			architectures:  []string{"ppc64le"},
			expectAffinity: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyArchitectureNodeAffinity(tt.pod, tt.architectures)

			if tt.expectAffinity {
				// Verify affinity structure was created
				if tt.pod.Spec.Affinity == nil {
					t.Fatal("Expected affinity to be created")
				}
				if tt.pod.Spec.Affinity.NodeAffinity == nil {
					t.Fatal("Expected node affinity to be created")
				}
				if tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
					t.Fatal("Expected required node affinity to be created")
				}

				// Verify architecture requirement was added
				found := false
				for _, term := range tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
					for _, expr := range term.MatchExpressions {
						if expr.Key == utils.ArchLabel && expr.Operator == corev1.NodeSelectorOpIn {
							found = true
							if len(expr.Values) != len(tt.architectures) {
								t.Errorf("Expected %d architectures, got %d", len(tt.architectures), len(expr.Values))
							}
							for i, arch := range tt.architectures {
								if expr.Values[i] != arch {
									t.Errorf("Expected architecture %s at index %d, got %s", arch, i, expr.Values[i])
								}
							}
						}
					}
				}
				if !found {
					t.Error("Architecture requirement not found in node affinity")
				}
			} else {
				// For empty architectures, affinity should not be modified
				if tt.pod.Spec.Affinity != nil {
					if tt.pod.Spec.Affinity.NodeAffinity != nil {
						if tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							if len(tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
								t.Error("Expected no node selector terms for empty architectures")
							}
						}
					}
				}
			}
		})
	}
}

func TestApplyArchitectureConstraints(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		architectures  []string
		expectModified bool
	}{
		{
			name: "remove old and apply new",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						utils.ArchLabel: "amd64",
					},
				},
			},
			architectures:  []string{"ppc64le"},
			expectModified: true,
		},
		{
			name: "apply to clean pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			architectures:  []string{"ppc64le"},
			expectModified: true,
		},
		{
			name: "empty architectures",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			architectures:  []string{},
			expectModified: false,
		},
		{
			name: "remove from both nodeSelector and nodeAffinity",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{
						utils.ArchLabel: "amd64",
					},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      utils.ArchLabel,
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"amd64"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			architectures:  []string{"ppc64le"},
			expectModified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := applyArchitectureConstraints(tt.pod, tt.architectures)

			if modified != tt.expectModified {
				t.Errorf("Expected modified=%v, got %v", tt.expectModified, modified)
			}

			if tt.expectModified && len(tt.architectures) > 0 {
				// Verify old constraints were removed
				if tt.pod.Spec.NodeSelector != nil {
					if _, exists := tt.pod.Spec.NodeSelector[utils.ArchLabel]; exists {
						t.Error("Old architecture constraint still exists in nodeSelector")
					}
				}

				// Verify new constraints were applied
				if tt.pod.Spec.Affinity == nil || tt.pod.Spec.Affinity.NodeAffinity == nil {
					t.Fatal("Expected node affinity to be created")
				}

				found := false
				if tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
					for _, term := range tt.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
						for _, expr := range term.MatchExpressions {
							if expr.Key == utils.ArchLabel {
								found = true
								if len(expr.Values) != len(tt.architectures) {
									t.Errorf("Expected %d architectures, got %d", len(tt.architectures), len(expr.Values))
								}
							}
						}
					}
				}
				if !found {
					t.Error("New architecture constraint not found")
				}
			}
		})
	}
}

func TestApplyArchitectureNodeAffinityPreservesOtherAffinity(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "test",
								},
							},
						},
					},
				},
			},
		},
	}

	applyArchitectureNodeAffinity(pod, []string{"ppc64le"})

	// Verify pod affinity was preserved
	if pod.Spec.Affinity.PodAffinity == nil {
		t.Error("Pod affinity was removed but should be preserved")
	}

	// Verify node affinity was added
	if pod.Spec.Affinity.NodeAffinity == nil {
		t.Error("Node affinity was not added")
	}
}
