package dependency

import appsv1 "k8s.io/api/apps/v1"

// FromDeployment extracts Secret and ConfigMap references exposed to regular or
// init containers through env/envFrom or mounted volumes. imagePullSecrets are
// deliberately excluded by product definition.
func FromDeployment(deployment *appsv1.Deployment) []Ref {
	if deployment == nil {
		return nil
	}
	namespace := deployment.Namespace
	pod := deployment.Spec.Template.Spec
	refs := make([]Ref, 0)

	appendContainerRefs := func(containers []coreContainer) {
		for _, c := range containers {
			for _, source := range c.envFrom {
				if source.secret != "" {
					refs = append(refs, Ref{Kind: Secret, Namespace: namespace, Name: source.secret})
				}
				if source.configMap != "" {
					refs = append(refs, Ref{Kind: ConfigMap, Namespace: namespace, Name: source.configMap})
				}
			}
			for _, source := range c.env {
				if source.secret != "" {
					refs = append(refs, Ref{Kind: Secret, Namespace: namespace, Name: source.secret})
				}
				if source.configMap != "" {
					refs = append(refs, Ref{Kind: ConfigMap, Namespace: namespace, Name: source.configMap})
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
			refs = append(refs, Ref{Kind: Secret, Namespace: namespace, Name: volume.Secret.SecretName})
		}
		if volume.ConfigMap != nil {
			refs = append(refs, Ref{Kind: ConfigMap, Namespace: namespace, Name: volume.ConfigMap.Name})
		}
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.Secret != nil {
					refs = append(refs, Ref{Kind: Secret, Namespace: namespace, Name: source.Secret.Name})
				}
				if source.ConfigMap != nil {
					refs = append(refs, Ref{Kind: ConfigMap, Namespace: namespace, Name: source.ConfigMap.Name})
				}
			}
		}
	}
	return Normalize(refs)
}
