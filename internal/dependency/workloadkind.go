package dependency

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ArgoRolloutAPIVersion is the API version of the Argo Rollouts CRD.
const ArgoRolloutAPIVersion = "argoproj.io/v1alpha1"

// ArgoRolloutGVK is the Argo Rollouts workload kind.
var ArgoRolloutGVK = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout"}

// WorkloadKind is a workload type KICK can discover dependencies for and
// restart.
type WorkloadKind struct {
	APIVersion string
	Kind       string
}

// GroupVersionKind returns the schema GVK for a workload kind.
func (w WorkloadKind) GroupVersionKind() schema.GroupVersionKind {
	gv, err := schema.ParseGroupVersion(w.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}
	}
	return gv.WithKind(w.Kind)
}

// ArgoRolloutWorkloadKind is the optional Argo Rollouts workload kind. It is
// only usable when the CRD is installed and the integration is enabled.
var ArgoRolloutWorkloadKind = WorkloadKind{APIVersion: ArgoRolloutAPIVersion, Kind: "Rollout"}

// IsArgoRollout reports whether an object reference points at an Argo Rollout.
func IsArgoRollout(apiVersion, kind string) bool {
	return apiVersion == ArgoRolloutAPIVersion && kind == "Rollout"
}

// NewArgoRolloutObject returns an empty typed-by-GVK object for Get/List calls.
func NewArgoRolloutObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(ArgoRolloutGVK)
	return obj
}

// rolloutPodTemplate returns the inline pod template of an Argo Rollout.
//
// A Rollout may instead point at a Deployment through spec.workloadRef. In that
// case it has no inline template and the referenced Deployment is discovered on
// its own, so nothing is extracted here.
func rolloutPodTemplate(obj *unstructured.Unstructured) (corev1.PodSpec, bool) {
	raw, found, err := unstructured.NestedMap(obj.Object, "spec", "template", "spec")
	if err != nil || !found {
		return corev1.PodSpec{}, false
	}
	var podSpec corev1.PodSpec
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, &podSpec); err != nil {
		return corev1.PodSpec{}, false
	}
	return podSpec, true
}

// extractUnstructuredDependencies extracts references from CRD-backed workloads
// that embed a standard pod template.
func extractUnstructuredDependencies(obj *unstructured.Unstructured) []DependencyRef {
	if obj.GroupVersionKind() != ArgoRolloutGVK {
		return nil
	}
	podSpec, ok := rolloutPodTemplate(obj)
	if !ok {
		return nil
	}
	return extractDependenciesFromPodSpec(obj.GetNamespace(), podSpec)
}
