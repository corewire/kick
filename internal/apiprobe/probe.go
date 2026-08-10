// Package apiprobe reports whether an optional API kind exists in the cluster.
//
// Registering a watch or a field index for a kind whose CRD is not installed
// makes the manager fail at startup, so every optional integration (Secrets
// Store CSI, Argo Rollouts, Kargo) is probed before it is wired up.
package apiprobe

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KindAvailable reports whether the cluster serves the given kind.
func KindAvailable(mapper meta.RESTMapper, gvk schema.GroupVersionKind) bool {
	if mapper == nil {
		return false
	}
	_, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	return err == nil
}
