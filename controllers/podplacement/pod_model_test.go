package podplacement

import (
	"context"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	. "github.com/onsi/gomega"

	mmoimage "github.com/openshift/multiarch-manager-operator/pkg/image"
	"github.com/openshift/multiarch-manager-operator/pkg/testing/image/fake"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"

	. "github.com/openshift/multiarch-manager-operator/pkg/testing/framework"
)

var ctx context.Context

func init() {
	ctx = context.TODO()
}

func TestPod_GetPodImagePullSecrets(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want []string
	}{
		{
			name: "pod with no imagePullSecrets",
			pod:  NewPod().Build(),
			want: []string{},
		},
		{
			name: "pod with imagePullSecrets",
			pod:  NewPod().WithImagePullSecrets("my-secret").Build(),
			want: []string{"my-secret"},
		},
		{
			name: "pod with empty imagePullSecrets",
			pod:  NewPod().WithImagePullSecrets().Build(),
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.GetPodImagePullSecrets()).To(Equal(tt.want))
		})
	}
}

func TestPod_HasSchedulingGate(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want bool
	}{
		{
			name: "pod with no scheduling gates",
			pod:  NewPod().Build(),
			want: false,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  NewPod().WithSchedulingGates().Build(),
			want: false,
		},
		{
			name: "pod with the multiarch-manager-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates(schedulingGateName).Build(),
			want: true,
		},
		{
			name: "pod with scheduling gates and NO multiarch-manager-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates("some-other-scheduling-gate").Build(),
			want: false,
		},
		{
			name: "pod with scheduling gates and the multiarch-manager-operator scheduling gate",
			pod: NewPod().WithSchedulingGates(
				"some-other-scheduling-gate-bar", schedulingGateName, "some-other-scheduling-gate-foo").Build(),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.HasSchedulingGate()).To(Equal(tt.want))
		})
	}
}

func TestPod_RemoveSchedulingGate(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want []v1.PodSchedulingGate
	}{
		{
			name: "pod with no scheduling gates",
			pod:  NewPod().Build(),
			want: nil,
		},
		{
			name: "pod with empty scheduling gates",
			pod:  NewPod().WithSchedulingGates().Build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with the multiarch-manager-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates(schedulingGateName).Build(),
			want: []v1.PodSchedulingGate{},
		},
		{
			name: "pod with scheduling gates and NO multiarch-manager-operator scheduling gate",
			pod:  NewPod().WithSchedulingGates("some-other-scheduling-gate").Build(),
			want: []v1.PodSchedulingGate{
				{
					Name: "some-other-scheduling-gate",
				},
			},
		},
		{
			name: "pod with scheduling gates and the multiarch-manager-operator scheduling gate",
			pod: NewPod().WithSchedulingGates(
				"some-other-scheduling-gate-bar", schedulingGateName,
				"some-other-scheduling-gate-foo").Build(),
			want: []v1.PodSchedulingGate{
				{
					Name: "some-other-scheduling-gate-bar",
				},
				{
					Name: "some-other-scheduling-gate-foo",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			pod.RemoveSchedulingGate()
			g := NewGomegaWithT(t)
			g.Expect(pod.Spec.SchedulingGates).To(Equal(tt.want))
		})
	}
}

func TestPod_imagesNamesSet(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want sets.Set[string]
	}{
		{
			name: "pod with a single container",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: "bar/foo:latest",
						},
					},
				},
			},
			want: sets.New[string]("//bar/foo:latest"),
		},
		{
			name: "pod with multiple containers, some with the same image",
			pod:  NewPod().WithContainersImages("bar/foo:latest", "bar/baz:latest", "bar/foo:latest").Build(),
			want: sets.New[string]("//bar/foo:latest", "//bar/baz:latest"),
		},
		{
			name: "pod with multiple containers and init containers",
			pod: NewPod().WithInitContainersImages("foo/bar:latest").WithContainersImages(
				"bar/foo:latest", "bar/baz:latest", "bar/foo:latest").Build(),
			want: sets.New[string]("//bar/foo:latest", "//bar/baz:latest", "//foo/bar:latest"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			g.Expect(pod.imagesNamesSet()).To(Equal(tt.want))
		})
	}
}

func TestPod_intersectImagesArchitecture(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		// pullSecretDataList is a list of pull secrets in the form of a slice of bytes. It is not used in the unit
		// tests. It is used in the integration tests.
		pullSecretDataList         [][]byte
		wantSupportedArchitectures sets.Set[string]
		wantErr                    bool
	}{
		{
			name:                       "pod with a single container and multi-arch image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with a single container and single-arch image",
			pod:                        NewPod().WithContainersImages(fake.SingleArchArm64Image).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers and same image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.MultiArchImage).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, single-arch image and multi-arch image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.SingleArchArm64Image).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, two multi-arch images",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, fake.MultiArchImage2).Build(),
			wantSupportedArchitectures: sets.New[string](utils.ArchitectureAmd64, utils.ArchitectureArm64),
		},
		{
			name:                       "pod with multiple containers, one non-existing image",
			pod:                        NewPod().WithContainersImages(fake.MultiArchImage, "non-existing-image").Build(),
			wantErr:                    true,
			wantSupportedArchitectures: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			gotSupportedArchitectures, err := pod.intersectImagesArchitecture(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(err).Should(WithTransform(func(err error) bool { return err != nil }, Equal(tt.wantErr)),
				"error expectation failed")
			g.Expect(gotSupportedArchitectures).Should(WithTransform(func(arches []string) sets.Set[string] {
				if arches == nil {
					return nil
				}
				return sets.New[string](arches...)
			}, Equal(tt.wantSupportedArchitectures)),
				"the set in gotSupportedArchitectures is not equal to the expected one")
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_getArchitecturePredicate(t *testing.T) {
	tests := []struct {
		name               string
		pod                v1.Pod
		pullSecretDataList [][]byte
		// Be aware that the values in the want.Values slice must be sorted alphabetically
		want    v1.NodeSelectorRequirement
		wantErr bool
	}{
		{
			name: "pod with several containers using multi-arch images",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: fake.MultiArchImage,
						},
					},
					InitContainers: []v1.Container{
						{
							Image: fake.MultiArchImage2,
						},
					},
				},
			},
			want: v1.NodeSelectorRequirement{
				Key:      utils.ArchLabel,
				Operator: v1.NodeSelectorOpIn,
				Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
			},
		},
		{
			name: "pod with non-existing image",
			pod: v1.Pod{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image: fake.MultiArchImage,
						},
						{
							Image: "non-existing-image",
						},
					},
				},
			},
			wantErr: true,
			want:    v1.NodeSelectorRequirement{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			got, err := pod.getArchitecturePredicate(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(err).Should(WithTransform(func(err error) bool { return err != nil }, Equal(tt.wantErr)),
				"error expectation failed")
			// sort the architectures to make the comparison easier
			sort.Strings(got.Values)
			g.Expect(got).To(Equal(tt.want))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_setArchNodeAffinity(t *testing.T) {
	tests := []struct {
		name string
		pod  v1.Pod
		want v1.Pod
	}{
		{
			name: "pod with empty node selector terms",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions().Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and nil match expressions",
			pod: NewPod().WithContainersImages(fake.SingleArchAmd64Image).WithNodeSelectorTermsMatchExpressions(
				nil).Build(),
			want: NewPod().WithContainersImages(fake.SingleArchAmd64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and empty match expressions",
			pod: NewPod().WithContainersImages(fake.SingleArchArm64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{}).Build(),
			want: NewPod().WithContainersImages(fake.SingleArchArm64Image).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and match expressions",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
				}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector terms and match expressions and an architecture requirement",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					},
				}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureS390x},
					},
				}, []v1.NodeSelectorRequirement{
					{
						Key:      "baz",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"foo"},
					}, {
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			g := NewGomegaWithT(t)
			pred, err := pod.getArchitecturePredicate(nil)
			g.Expect(err).ShouldNot(HaveOccurred())
			pod.setArchNodeAffinity(pred)
			g.Expect(pod.Spec.Affinity).Should(Equal(tt.want.Spec.Affinity))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}

func TestPod_SetNodeAffinityArchRequirement(t *testing.T) {
	tests := []struct {
		name               string
		pullSecretDataList [][]byte
		pod                v1.Pod
		want               v1.Pod
	}{
		{
			name: "pod with no node selector terms",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(nil).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				},
			).Build(),
		},
		{
			name: "pod with node selector and no architecture requirement",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar").Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors(
				"foo", "bar").WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with node selector and architecture requirement",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar",
				utils.ArchLabel, utils.ArchitectureArm64).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectors("foo", "bar",
				utils.ArchLabel, utils.ArchitectureArm64).Build(),
		},
		{
			name: "pod with no affinity",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with no node affinity",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(nil).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with no required during scheduling ignored during execution",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage).WithNodeAffinity().Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "pod with predefined node selector terms in the required during scheduling ignored during execution",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions([]v1.NodeSelectorRequirement{
				{
					Key:      "foo",
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{"bar"},
				},
			}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      "foo",
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "other affinity types should not be modified",
			pod: NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(&v1.Affinity{
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
						{
							TopologyKey: "foo",
						},
					},
				},
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
						{
							Weight: 1,
							Preference: v1.NodeSelectorTerm{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "foo",
										Operator: v1.NodeSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			}).Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage).WithAffinity(&v1.Affinity{
				PodAffinity: &v1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
						{
							TopologyKey: "foo",
						},
					},
				},
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
						{
							Weight: 1,
							Preference: v1.NodeSelectorTerm{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "foo",
										Operator: v1.NodeSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			}).WithNodeSelectorTermsMatchExpressions(
				[]v1.NodeSelectorRequirement{
					{
						Key:      utils.ArchLabel,
						Operator: v1.NodeSelectorOpIn,
						Values:   []string{utils.ArchitectureAmd64, utils.ArchitectureArm64},
					},
				}).Build(),
		},
		{
			name: "should not modify the pod if unable to inspect the images",
			pod:  NewPod().WithContainersImages(fake.MultiArchImage, "non-readable-image").Build(),
			want: NewPod().WithContainersImages(fake.MultiArchImage, "non-readable-image").Build(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageInspectionCache = fake.FacadeSingleton()
			pod := &Pod{
				Pod: tt.pod,
				ctx: ctx,
			}
			pod.SetNodeAffinityArchRequirement(tt.pullSecretDataList)
			g := NewGomegaWithT(t)
			g.Expect(pod.Spec.Affinity).Should(Equal(tt.want.Spec.Affinity))
			imageInspectionCache = mmoimage.FacadeSingleton()
		})
	}
}
