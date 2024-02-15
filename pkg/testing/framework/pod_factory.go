package framework

import (
	//#nosec G505: Blocklisted import crypto/sha1: weak cryptographic primitive
	"crypto/sha1"
	"encoding/hex"

	v1 "k8s.io/api/core/v1"
)

// PodFactory is a builder for v1.Pod objects to be used only in unit tests.
type PodFactory struct {
	pod v1.Pod
}

// NewPod returns a new PodFactory to build v1.Pod objects. It is meant to be used only in unit tests.
func NewPod() *PodFactory {
	return &PodFactory{
		pod: v1.Pod{},
	}
}

func (p *PodFactory) WithImagePullSecrets(imagePullSecrets ...string) *PodFactory {
	p.pod.Spec.ImagePullSecrets = make([]v1.LocalObjectReference, len(imagePullSecrets))
	for i, secret := range imagePullSecrets {
		p.pod.Spec.ImagePullSecrets[i] = v1.LocalObjectReference{
			Name: secret,
		}
	}
	return p
}

func (p *PodFactory) WithSchedulingGates(schedulingGates ...string) *PodFactory {
	p.pod.Spec.SchedulingGates = make([]v1.PodSchedulingGate, len(schedulingGates))
	for i, gate := range schedulingGates {
		p.pod.Spec.SchedulingGates[i] = v1.PodSchedulingGate{
			Name: gate,
		}
	}
	return p
}

func (p *PodFactory) WithContainersImages(images ...string) *PodFactory {
	p.pod.Spec.Containers = make([]v1.Container, len(images))
	for i, image := range images {
		// compute hash of the image name
		//#nosec G401: Use of weak cryptographic primitive
		sha := sha1.New()
		sha.Write([]byte(image))
		p.pod.Spec.Containers[i] = v1.Container{
			Image: image,
			Name:  hex.EncodeToString(sha.Sum(nil)), // hash of the image name (40 characters, 63 is max)
		}
	}
	return p
}

func (p *PodFactory) WithInitContainersImages(images ...string) *PodFactory {
	p.pod.Spec.InitContainers = make([]v1.Container, len(images))
	for i, image := range images {
		p.pod.Spec.InitContainers[i] = v1.Container{
			Image: image,
		}
	}
	return p
}

// WithAffinity adds the affinity to the pod. If initialAffinity is not nil, it is used as the initial value
// of the pod's affinity. Otherwise, the pod's affinity is initialized to an empty affinity if it is nil.
func (p *PodFactory) WithAffinity(initialAffinity *v1.Affinity) *PodFactory {
	if p.pod.Spec.Affinity == nil {
		p.pod.Spec.Affinity = &v1.Affinity{}
	}
	if initialAffinity != nil {
		p.pod.Spec.Affinity = initialAffinity
	}
	return p
}

func (p *PodFactory) WithNodeAffinity() *PodFactory {
	p.WithAffinity(nil)
	if p.pod.Spec.Affinity.NodeAffinity == nil {
		p.pod.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{}
	}
	return p
}

func (p *PodFactory) WithRequiredDuringSchedulingIgnoredDuringExecution() *PodFactory {
	p.WithNodeAffinity()
	if p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{}
	}
	return p
}

func (p *PodFactory) WithNodeSelectorTermsMatchExpressions(
	nodeSelectorTermsMatchExpressions ...[]v1.NodeSelectorRequirement) *PodFactory {
	p.WithRequiredDuringSchedulingIgnoredDuringExecution()
	p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = make(
		[]v1.NodeSelectorTerm, len(nodeSelectorTermsMatchExpressions))
	for i, matchExpressions := range nodeSelectorTermsMatchExpressions {
		p.pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i] =
			v1.NodeSelectorTerm{
				MatchExpressions: matchExpressions,
			}
	}
	return p
}

func (p *PodFactory) WithNodeSelectors(kv ...string) *PodFactory {
	if p.pod.Spec.NodeSelector == nil {
		p.pod.Spec.NodeSelector = make(map[string]string)
	}
	if len(kv)%2 != 0 {
		panic("the number of arguments must be even")
	}
	for i := 0; i < len(kv); i += 2 {
		p.pod.Spec.NodeSelector[kv[i]] = kv[i+1]
	}
	return p
}

func (p *PodFactory) WithGenerateName(name string) *PodFactory {
	p.pod.GenerateName = name
	return p
}

func (p *PodFactory) WithNamespace(namespace string) *PodFactory {
	p.pod.Namespace = namespace
	return p
}

func (p *PodFactory) Build() v1.Pod {
	return p.pod
}
