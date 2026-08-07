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

	appendContainerRefs := func(containers []coreContainer) {
		for _, c := range containers {
			for _, source := range c.envFrom {
				if source.secret != "" {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: Secret, Namespace: namespace, Name: source.secret})
				}
				if source.configMap != "" {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: namespace, Name: source.configMap})
				}
			}
			for _, source := range c.env {
				if source.secret != "" {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: Secret, Namespace: namespace, Name: source.secret})
				}
				if source.configMap != "" {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: namespace, Name: source.configMap})
				}
			}
		}
	}

	regular := make([]coreContainer, 0, len(pod.Containers))
	for _, c := range pod.Containers {
		regular = append(regular, adaptContainer(c))
	}
	init := make([]coreContainer, 0, len(pod.InitContainers))
	for _, c := range pod.InitContainers {
		init = append(init, adaptContainer(c))
	}
	appendContainerRefs(regular)
	appendContainerRefs(init)

	for _, volume := range pod.Volumes {
		if volume.Secret != nil {
			refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: Secret, Namespace: namespace, Name: volume.Secret.SecretName})
		}
		if volume.ConfigMap != nil {
			refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: namespace, Name: volume.ConfigMap.Name})
		}
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.Secret != nil {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: Secret, Namespace: namespace, Name: source.Secret.Name})
				}
				if source.ConfigMap != nil {
					refs = append(refs, DependencyRef{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: namespace, Name: source.ConfigMap.Name})
				}
			}
		}
	}
	return Normalize(refs)
}

// FromDeployment is a compatibility wrapper retained for existing callers.
func FromDeployment(deployment *appsv1.Deployment) []Ref {
	return ExtractDependencies(deployment)
}
