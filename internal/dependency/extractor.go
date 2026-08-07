package dependency

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtractDependencies extracts Secret and ConfigMap references exposed to
// regular/init containers through env/envFrom and mounted/projected volumes.
// imagePullSecrets are deliberately excluded by product definition.
func ExtractDependencies(deployment *appsv1.Deployment) []DependencyRef {
	if deployment == nil {
		return nil
	}
	return extractDependenciesFromPodSpec(deployment.Namespace, deployment.Spec.Template.Spec)
}

// ExtractStatefulSetDependencies extracts references consumed by a StatefulSet.
func ExtractStatefulSetDependencies(statefulSet *appsv1.StatefulSet) []DependencyRef {
	if statefulSet == nil {
		return nil
	}
	return extractDependenciesFromPodSpec(statefulSet.Namespace, statefulSet.Spec.Template.Spec)
}

// ExtractDaemonSetDependencies extracts references consumed by a DaemonSet.
func ExtractDaemonSetDependencies(daemonSet *appsv1.DaemonSet) []DependencyRef {
	if daemonSet == nil {
		return nil
	}
	return extractDependenciesFromPodSpec(daemonSet.Namespace, daemonSet.Spec.Template.Spec)
}

// ExtractDependenciesForObject dispatches extraction based on workload type.
func ExtractDependenciesForObject(obj client.Object) []DependencyRef {
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		return ExtractDependencies(workload)
	case *appsv1.StatefulSet:
		return ExtractStatefulSetDependencies(workload)
	case *appsv1.DaemonSet:
		return ExtractDaemonSetDependencies(workload)
	default:
		return nil
	}
}

func extractDependenciesFromPodSpec(namespace string, pod corev1.PodSpec) []DependencyRef {
	refs := make([]DependencyRef, 0)
	refs = append(refs, containerRefs(namespace, pod.Containers)...)
	refs = append(refs, containerRefs(namespace, pod.InitContainers)...)
	refs = append(refs, volumeRefs(namespace, pod.Volumes)...)
	return Normalize(refs)
}

// appendNamed appends a dependency ref unless the referenced name is empty.
func appendNamed(refs []DependencyRef, namespace string, kind Kind, name string) []DependencyRef {
	if name == "" {
		return refs
	}
	return append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: kind, Namespace: namespace, Name: name})
}

// containerRefs collects Secret/ConfigMap refs exposed through env and envFrom.
func containerRefs(namespace string, containers []corev1.Container) []DependencyRef {
	refs := make([]DependencyRef, 0)
	for _, container := range containers {
		adapted := adaptContainer(container)
		for _, source := range adapted.envFrom {
			refs = appendNamed(refs, namespace, Secret, source.secret)
			refs = appendNamed(refs, namespace, ConfigMap, source.configMap)
		}
		for _, source := range adapted.env {
			refs = appendNamed(refs, namespace, Secret, source.secret)
			refs = appendNamed(refs, namespace, ConfigMap, source.configMap)
		}
	}
	return refs
}

// volumeRefs collects Secret/ConfigMap refs from mounted and projected volumes.
func volumeRefs(namespace string, volumes []corev1.Volume) []DependencyRef {
	refs := make([]DependencyRef, 0)
	for _, volume := range volumes {
		if volume.Secret != nil {
			refs = appendNamed(refs, namespace, Secret, volume.Secret.SecretName)
		}
		if volume.ConfigMap != nil {
			refs = appendNamed(refs, namespace, ConfigMap, volume.ConfigMap.Name)
		}
		if volume.Projected == nil {
			continue
		}
		for _, source := range volume.Projected.Sources {
			if source.Secret != nil {
				refs = appendNamed(refs, namespace, Secret, source.Secret.Name)
			}
			if source.ConfigMap != nil {
				refs = appendNamed(refs, namespace, ConfigMap, source.ConfigMap.Name)
			}
		}
	}
	return refs
}

// FromDeployment is a compatibility wrapper retained for existing callers.
func FromDeployment(deployment *appsv1.Deployment) []Ref {
	return ExtractDependencies(deployment)
}
