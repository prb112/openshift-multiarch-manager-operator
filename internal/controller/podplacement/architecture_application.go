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
	corev1 "k8s.io/api/core/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"
)

// applyArchitectureNodeAffinity applies new architecture constraints to a pod's node affinity
// by updating in-place the requiredDuringSchedulingIgnoredDuringExecution matchExpressions for
// kubernetes.io/arch using the In operator.
// It initializes the affinity structure if needed and merges the architecture constraint
// into existing terms to avoid Kubernetes rejecting updates due to NodeSelectorTerms modifications.
// The architectures list must not be empty.
//
// This function updates architecture constraints in-place by:
// 1. Removing any existing kubernetes.io/arch matchExpressions from all terms
// 2. Adding the new architecture requirement to each term (or creating a new term if none exist)
// This approach avoids the Kubernetes API rejection: "no additions/deletions to non-empty NodeSelectorTerms list are allowed"
func applyArchitectureNodeAffinity(pod *corev1.Pod, architectures []string) {
	log := ctrllog.Log.WithName("podplacement")

	if len(architectures) == 0 {
		// Should not happen, but guard against it
		return
	}

	// Initialize affinity structure if needed
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}

	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{}
	}

	// Create the architecture requirement
	architectureRequirement := corev1.NodeSelectorRequirement{
		Key:      utils.ArchLabel,
		Operator: corev1.NodeSelectorOpIn,
		Values:   architectures,
	}

	existingTerms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms

	log.Info("Before applyArchitectureNodeAffinity",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"termCount", len(existingTerms),
		"terms", existingTerms)

	// If no terms exist, create a single architecture-only term
	if len(existingTerms) == 0 {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{architectureRequirement},
			},
		}
		return
	}

	// Update in-place: remove old arch constraints and add new ones to each term
	// This preserves the NodeSelectorTerms array structure to avoid Kubernetes API rejection
	for i := range existingTerms {
		// Remove any existing architecture matchExpressions from this term
		var cleanedExpressions []corev1.NodeSelectorRequirement
		for _, expr := range existingTerms[i].MatchExpressions {
			if expr.Key != utils.ArchLabel {
				cleanedExpressions = append(cleanedExpressions, expr)
			}
		}

		// Add the new architecture requirement
		cleanedExpressions = append(cleanedExpressions, architectureRequirement)
		existingTerms[i].MatchExpressions = cleanedExpressions
	}

	log.Info("After applyArchitectureNodeAffinity",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"termCount", len(existingTerms),
		"terms", existingTerms)
}

// applyArchitectureConstraints applies architecture constraints to a pod by updating
// the node affinity in-place. It also removes architecture from nodeSelector if present.
// This ensures the plugin's architecture selection takes full effect while avoiding
// Kubernetes API rejections for NodeSelectorTerms modifications.
// Returns true if any changes were made to the pod.
func applyArchitectureConstraints(pod *corev1.Pod, architectures []string) bool {
	log := ctrllog.Log.WithName("podplacement")

	if len(architectures) == 0 {
		return false
	}

	log.Info("Applying architecture constraints",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"architectures", architectures)

	// Remove architecture from nodeSelector (this is safe and doesn't cause API rejections)
	removedFromNodeSelector := removeArchitectureFromNodeSelector(pod)

	// Update architecture constraints in-place within node affinity
	applyArchitectureNodeAffinity(pod, architectures)

	return removedFromNodeSelector || true // Always return true if we applied architectures
}
