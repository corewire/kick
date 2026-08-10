package dependency

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const coreAPIVersion = "v1"

// SecretsStoreAPIVersion is the API version of the Secrets Store CSI driver CRDs.
const SecretsStoreAPIVersion = "secrets-store.csi.x-k8s.io/v1"

// SecretProviderClassGVK identifies the Secrets Store CSI SecretProviderClass.
var SecretProviderClassGVK = schema.GroupVersionKind{
	Group:   "secrets-store.csi.x-k8s.io",
	Version: "v1",
	Kind:    "SecretProviderClass",
}

// NewSecretProviderClassObject returns an empty object typed by GVK for
// Get/List calls against the optional Secrets Store CSI CRD.
func NewSecretProviderClassObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(SecretProviderClassGVK)
	return obj
}

// SecretsStoreCSIDriver is the CSI driver name registered by the Secrets Store
// CSI driver. It deliberately differs from the API group: the driver registers
// as "secrets-store.csi.k8s.io" while its CRDs live in
// "secrets-store.csi.x-k8s.io".
const SecretsStoreCSIDriver = "secrets-store.csi.k8s.io"

// secretProviderClassVolumeAttribute is the CSI volume attribute naming the
// SecretProviderClass a pod mounts.
const secretProviderClassVolumeAttribute = "secretProviderClass"

// Kind is a supported runtime dependency kind.
type Kind string

const (
	Secret    Kind = "Secret"
	ConfigMap Kind = "ConfigMap"
	// SecretProviderClass is an external secret mounted through the Secrets Store
	// CSI driver. Its content lives outside the cluster, so freshness is derived
	// from SecretProviderClassPodStatus instead of the object itself.
	SecretProviderClass Kind = "SecretProviderClass"
)

// DependencyRef is a namespaced Secret or ConfigMap consumed by a workload.
// Identity is apiVersion+kind+namespace+name.
type DependencyRef struct {
	APIVersion string
	Kind       Kind
	Namespace  string
	Name       string
}

// Ref is kept as an alias for compatibility with in-progress tasks.
type Ref = DependencyRef

// Normalize removes duplicates and returns deterministic ordering.
func Normalize(in []DependencyRef) []DependencyRef {
	seen := make(map[DependencyRef]struct{}, len(in))
	out := make([]DependencyRef, 0, len(in))
	for _, ref := range in {
		if ref.APIVersion == "" || ref.Name == "" || ref.Namespace == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].APIVersion != out[j].APIVersion {
			return out[i].APIVersion < out[j].APIVersion
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}
