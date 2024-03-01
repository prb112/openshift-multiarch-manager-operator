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

package podplacement

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/multiarch-manager-operator/pkg/image"
	"github.com/openshift/multiarch-manager-operator/pkg/utils"
)

var (
	// imageInspectionCache is the facade singleton used to inspect images. It is defined here to facilitate testing.
	imageInspectionCache image.ICache = image.FacadeSingleton()
)

type Pod struct {
	corev1.Pod
	ctx context.Context
}

func (pod *Pod) GetPodImagePullSecrets() []string {
	if pod.Spec.ImagePullSecrets == nil {
		// If the imagePullSecrets array is nil, return emptylist
		return []string{}
	}
	var secretRefs = make([]string, len(pod.Spec.ImagePullSecrets))
	for i, secret := range pod.Spec.ImagePullSecrets {
		secretRefs[i] = secret.Name
	}
	return secretRefs
}

func (pod *Pod) HasSchedulingGate() bool {
	if pod.Spec.SchedulingGates == nil {
		// If the schedulingGates array is nil, we return false
		return false
	}
	for _, condition := range pod.Spec.SchedulingGates {
		if condition.Name == schedulingGateName {
			return true
		}
	}
	// the scheduling gate is not found.
	return false
}

func (pod *Pod) RemoveSchedulingGate() {
	if len(pod.Spec.SchedulingGates) == 0 {
		// If the schedulingGates array is nil, we return
		return
	}
	filtered := make([]corev1.PodSchedulingGate, 0, len(pod.Spec.SchedulingGates))
	for _, schedulingGate := range pod.Spec.SchedulingGates {
		if schedulingGate.Name != schedulingGateName {
			filtered = append(filtered, schedulingGate)
		}
	}
	pod.Spec.SchedulingGates = filtered
	// The scheduling gate is removed. We also add a label to the pod to indicate that the scheduling gate was removed
	// and this pod was processed by the operator. That's useful for testing and debugging, but also gives the user
	// an indication that the pod was processed by the operator.
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[utils.SchedulingGateLabel] = utils.SchedulingGateLabelValueRemoved
}

// SetNodeAffinityArchRequirement wraps the logic to set the nodeAffinity for the pod.
// It verifies first that no nodeSelector field is set for the kubernetes.io/arch label.
// Then, it computes the intersection of the architectures supported by the images used by the pod via pod.getArchitecturePredicate.
// Finally, it initializes the nodeAffinity for the pod and set it to the computed requirement via the pod.setArchNodeAffinity method.
func (pod *Pod) SetNodeAffinityArchRequirement(pullSecretDataList [][]byte) {
	log := ctrllog.FromContext(pod.ctx)

	if pod.Spec.NodeSelector != nil {
		for key := range pod.Spec.NodeSelector {
			if key == utils.ArchLabel {
				// if the pod has the nodeSelector field set for the kubernetes.io/arch label, we ignore it.
				// in fact, the nodeSelector field is ANDed with the nodeAffinity field, and we want to give the user the main control, if they
				// manually set a predicate for the kubernetes.io/arch label.
				// The same behavior is implemented below within each
				// nodeSelectorTerm's MatchExpressions field.
				log.V(3).Info("The pod has the nodeSelector field set for the kubernetes.io/arch label. Ignoring the pod...")
				return
			}
		}
	}

	requirement, err := pod.getArchitecturePredicate(pullSecretDataList)
	if err != nil {
		log.Error(err, "Error getting the architecture predicate. The pod will not have the nodeAffinity set.")
		return
	}

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	pod.setArchNodeAffinity(requirement)
}

// setArchNodeAffinity sets the node affinity for the pod to the given requirement based on the rules in
// the sig-scheduling's KEP-3838: https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/3838-pod-mutable-scheduling-directives.
func (pod *Pod) setArchNodeAffinity(requirement corev1.NodeSelectorRequirement) {
	// the .requiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms are ORed
	if len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
		// We create a new array of NodeSelectorTerm of length 1 so that we can always iterate it in the next.
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = make([]corev1.NodeSelectorTerm, 1)
	}
	nodeSelectorTerms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	// The expressions within the nodeSelectorTerms are ANDed.
	// Therefore, we iterate over the nodeSelectorTerms and add an expression to each of the terms to verify the
	// kubernetes.io/arch label has compatible values.
	// Note that the NodeSelectorTerms will always be long at least 1, because we (re-)created it with size 1 above if it was nil (or having 0 length).
	var skipMatchExpressionPatch bool
	var patched bool
	for i := range nodeSelectorTerms {
		skipMatchExpressionPatch = false
		if nodeSelectorTerms[i].MatchExpressions == nil {
			nodeSelectorTerms[i].MatchExpressions = make([]corev1.NodeSelectorRequirement, 0, 1)
		}
		// Check if the nodeSelectorTerm already has a matchExpression for the kubernetes.io/arch label.
		// if yes, we ignore to add it.
		for _, expression := range nodeSelectorTerms[i].MatchExpressions {
			if expression.Key == requirement.Key {
				skipMatchExpressionPatch = true
				break
			}
		}
		// if skipMatchExpressionPatch is true, we skip to add the matchExpression so that conflictual matchExpressions provided by the user are not overwritten.
		if !skipMatchExpressionPatch {
			nodeSelectorTerms[i].MatchExpressions = append(nodeSelectorTerms[i].MatchExpressions, requirement)
			patched = true
		}
	}
	// if the nodeSelectorTerms were patched at least once, we set the nodeAffinity label to the set value, to keep
	// track of the fact that the nodeAffinity was patched by the operator.
	if patched {
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		pod.Labels[utils.NodeAffinityLabel] = utils.NodeAffinityLabelValueSet
	}
}

func (pod *Pod) getArchitecturePredicate(pullSecretDataList [][]byte) (corev1.NodeSelectorRequirement, error) {
	architectures, err := pod.intersectImagesArchitecture(pullSecretDataList)
	// if an error occurs, we return an empty NodeSelectorRequirement and the error.
	if err != nil {
		return corev1.NodeSelectorRequirement{}, err
	}
	return corev1.NodeSelectorRequirement{
		Key:      utils.ArchLabel,
		Operator: corev1.NodeSelectorOpIn,
		Values:   architectures,
	}, nil
}

func (pod *Pod) imagesNamesSet() sets.Set[string] {
	imageNamesSet := sets.New[string]()
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		imageNamesSet.Insert(fmt.Sprintf("//%s", container.Image))
	}
	return imageNamesSet
}

// inspect returns the list of supported architectures for the images used by the pod.
// if an error occurs, it returns the error and a nil slice of strings.
func (pod *Pod) intersectImagesArchitecture(pullSecretDataList [][]byte) (supportedArchitectures []string, err error) {
	log := ctrllog.FromContext(pod.ctx)
	imageNamesSet := pod.imagesNamesSet()
	log.V(3).Info("Images list for pod", "imageNamesSet", fmt.Sprintf("%+v", imageNamesSet))
	// https://github.com/containers/skopeo/blob/v1.11.1/cmd/skopeo/inspect.go#L72
	// Iterate over the images, get their architectures and intersect (as in set intersection) them each other
	var supportedArchitecturesSet sets.Set[string]
	for imageName := range imageNamesSet {
		log.V(5).Info("Checking image", "imageName", imageName)
		currentImageSupportedArchitectures, err := imageInspectionCache.GetCompatibleArchitecturesSet(pod.ctx, imageName, pullSecretDataList)
		if err != nil {
			log.V(3).Error(err, "Error inspecting the image", "imageName", imageName)
			return nil, err
		}
		if supportedArchitecturesSet == nil {
			supportedArchitecturesSet = currentImageSupportedArchitectures
		} else {
			supportedArchitecturesSet = supportedArchitecturesSet.Intersection(currentImageSupportedArchitectures)
		}
	}
	return sets.List(supportedArchitecturesSet), nil
}
