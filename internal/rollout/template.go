package rollout

import (
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

const podTemplateHashLabel = "pod-template-hash"

// TemplatesEquivalent compares deployment and replicaset pod templates while
// ignoring hash-only labels used by rollout controller bookkeeping.
func TemplatesEquivalent(deploymentTemplate, replicaSetTemplate corev1.PodTemplateSpec) bool {
	left := deploymentTemplate.DeepCopy()
	right := replicaSetTemplate.DeepCopy()

	stripHashLabel(left)
	stripHashLabel(right)

	return apiequality.Semantic.DeepEqual(*left, *right)
}

func stripHashLabel(template *corev1.PodTemplateSpec) {
	if template.Labels == nil {
		return
	}
	delete(template.Labels, podTemplateHashLabel)
}
