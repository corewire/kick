package dependency

import (
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// #nosec G101 -- these are field-index keys, not credentials.
	SecretReferenceIndexField    = "kick.corewire.io/secretReferences"
	ConfigMapReferenceIndexField = "kick.corewire.io/configMapReferences"
	// #nosec G101 -- this is a field-index key, not a credential.
	SecretProviderClassReferenceIndexField = "kick.corewire.io/secretProviderClassReferences"
)

// ConsumerTarget identifies one workload that consumes a dependency source.
type ConsumerTarget struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

// RegisterDeploymentReverseIndexes installs Deployment field indexes for source
// reverse lookups. Index values are <namespace>/<name>.
func RegisterDeploymentReverseIndexes(ctx context.Context, indexer client.FieldIndexer) error {
	return RegisterWorkloadReverseIndexes(ctx, indexer)
}

// RegisterWorkloadReverseIndexes installs field indexes for the built-in
// workload kinds plus any optional CRD-backed kinds. Index values are
// <namespace>/<name>.
//
// Only pass optional kinds whose CRD is installed: controller-runtime fails at
// startup when an index is registered for an unknown kind.
func RegisterWorkloadReverseIndexes(ctx context.Context, indexer client.FieldIndexer, optional ...WorkloadKind) error {
	indexByKind := func(kind Kind) client.IndexerFunc {
		return func(raw client.Object) []string {
			refs := ExtractDependenciesForObject(raw)
			out := make([]string, 0, len(refs))
			for _, ref := range refs {
				if ref.Kind == kind {
					out = append(out, ref.Namespace+"/"+ref.Name)
				}
			}
			return out
		}
	}

	fields := []struct {
		name string
		fn   client.IndexerFunc
	}{
		{SecretReferenceIndexField, indexByKind(Secret)},
		{ConfigMapReferenceIndexField, indexByKind(ConfigMap)},
		{SecretProviderClassReferenceIndexField, indexByKind(SecretProviderClass)},
	}

	for _, obj := range indexableObjects(optional) {
		for _, field := range fields {
			if err := indexer.IndexField(ctx, obj, field.name, field.fn); err != nil {
				return err
			}
		}
	}

	return nil
}

// indexableObjects returns one prototype object per indexed workload kind.
func indexableObjects(optional []WorkloadKind) []client.Object {
	objects := []client.Object{&appsv1.Deployment{}, &appsv1.StatefulSet{}, &appsv1.DaemonSet{}}
	for _, kind := range optional {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(kind.GroupVersionKind())
		objects = append(objects, obj)
	}
	return objects
}

// LookupConsumingDeployments returns deployment keys consuming a given source.
func LookupConsumingDeployments(ctx context.Context, c client.Client, ref DependencyRef) ([]ConsumerTarget, error) {
	all, err := LookupConsumingWorkloads(ctx, c, ref)
	if err != nil {
		return nil, err
	}
	out := make([]ConsumerTarget, 0, len(all))
	for _, target := range all {
		if target.Kind == "Deployment" {
			out = append(out, target)
		}
	}
	return out, nil
}

// LookupConsumingWorkloads returns all workload kinds consuming a given source
// reference. Optional kinds must match the ones passed to
// RegisterWorkloadReverseIndexes, otherwise the lookup fails on a missing index.
func LookupConsumingWorkloads(ctx context.Context, c client.Client, ref DependencyRef, optional ...WorkloadKind) ([]ConsumerTarget, error) {
	field := indexFieldFor(ref.Kind)
	if field == "" {
		return nil, nil
	}
	match := client.MatchingFields{field: ref.Namespace + "/" + ref.Name}

	var deployments appsv1.DeploymentList
	if err := c.List(ctx, &deployments, client.InNamespace(ref.Namespace), match); err != nil {
		return nil, err
	}
	var statefulSets appsv1.StatefulSetList
	if err := c.List(ctx, &statefulSets, client.InNamespace(ref.Namespace), match); err != nil {
		return nil, err
	}
	var daemonSets appsv1.DaemonSetList
	if err := c.List(ctx, &daemonSets, client.InNamespace(ref.Namespace), match); err != nil {
		return nil, err
	}

	targets := make([]ConsumerTarget, 0, len(deployments.Items)+len(statefulSets.Items)+len(daemonSets.Items))
	for _, deployment := range deployments.Items {
		targets = append(targets, ConsumerTarget{APIVersion: "apps/v1", Kind: "Deployment", Namespace: deployment.Namespace, Name: deployment.Name})
	}
	for _, statefulSet := range statefulSets.Items {
		targets = append(targets, ConsumerTarget{APIVersion: "apps/v1", Kind: "StatefulSet", Namespace: statefulSet.Namespace, Name: statefulSet.Name})
	}
	for _, daemonSet := range daemonSets.Items {
		targets = append(targets, ConsumerTarget{APIVersion: "apps/v1", Kind: "DaemonSet", Namespace: daemonSet.Namespace, Name: daemonSet.Name})
	}

	for _, kind := range optional {
		var list unstructured.UnstructuredList
		list.SetGroupVersionKind(kind.GroupVersionKind().GroupVersion().WithKind(kind.Kind + "List"))
		if err := c.List(ctx, &list, client.InNamespace(ref.Namespace), match); err != nil {
			return nil, err
		}
		for i := range list.Items {
			targets = append(targets, ConsumerTarget{APIVersion: kind.APIVersion, Kind: kind.Kind, Namespace: list.Items[i].GetNamespace(), Name: list.Items[i].GetName()})
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		if targets[i].Namespace != targets[j].Namespace {
			return targets[i].Namespace < targets[j].Namespace
		}
		return targets[i].Name < targets[j].Name
	})
	return targets, nil
}

func indexFieldFor(kind Kind) string {
	switch kind {
	case Secret:
		return SecretReferenceIndexField
	case ConfigMap:
		return ConfigMapReferenceIndexField
	case SecretProviderClass:
		return SecretProviderClassReferenceIndexField
	default:
		return ""
	}
}
