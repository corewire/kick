package dependency

import corev1 "k8s.io/api/core/v1"

type namedSource struct{ secret, configMap string }
type coreContainer struct{ envFrom, env []namedSource }

func adaptContainer(c corev1.Container) coreContainer {
	out := coreContainer{}
	for _, source := range c.EnvFrom {
		item := namedSource{}
		if source.SecretRef != nil {
			item.secret = source.SecretRef.Name
		}
		if source.ConfigMapRef != nil {
			item.configMap = source.ConfigMapRef.Name
		}
		out.envFrom = append(out.envFrom, item)
	}
	for _, variable := range c.Env {
		if variable.ValueFrom == nil {
			continue
		}
		item := namedSource{}
		if variable.ValueFrom.SecretKeyRef != nil {
			item.secret = variable.ValueFrom.SecretKeyRef.Name
		}
		if variable.ValueFrom.ConfigMapKeyRef != nil {
			item.configMap = variable.ValueFrom.ConfigMapKeyRef.Name
		}
		out.env = append(out.env, item)
	}
	return out
}
