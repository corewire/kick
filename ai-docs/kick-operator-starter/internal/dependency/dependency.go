package dependency

import "sort"

// Kind is a supported runtime dependency kind.
type Kind string

const (
	Secret    Kind = "Secret"
	ConfigMap Kind = "ConfigMap"
)

// Ref is a namespaced Secret or ConfigMap consumed by a workload.
type Ref struct {
	Kind      Kind
	Namespace string
	Name      string
}

// Normalize removes duplicates and returns deterministic ordering.
func Normalize(in []Ref) []Ref {
	seen := make(map[Ref]struct{}, len(in))
	out := make([]Ref, 0, len(in))
	for _, ref := range in {
		if ref.Name == "" || ref.Namespace == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool {
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
