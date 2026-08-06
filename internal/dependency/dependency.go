package dependency

import "sort"

const coreAPIVersion = "v1"

// Kind is a supported runtime dependency kind.
type Kind string

const (
	Secret    Kind = "Secret"
	ConfigMap Kind = "ConfigMap"
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
