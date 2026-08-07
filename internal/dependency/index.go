package dependency

import (
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// #nosec G101 -- these are field-index keys, not credentials.
	SecretReferenceIndexField    = "kick.corewire.io/secretReferences"
	ConfigMapReferenceIndexField = "kick.corewire.io/configMapReferences"
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
	if err := RegisterWorkloadReverseIndexes(ctx, indexer); err != nil {
		return err
	}
	return nil
}

// RegisterWorkloadReverseIndexes installs field indexes for all supported
// workload kinds. Index values are <namespace>/<name>.
func RegisterWorkloadReverseIndexes(ctx context.Context, indexer client.FieldIndexer) error {
	indexSecret := func(raw client.Object) []string {
		refs := ExtractDependenciesForObject(raw)
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.Kind == Secret {
				out = append(out, ref.Namespace+"/"+ref.Name)
			}
		}
		return out
	}
	indexConfigMap := func(raw client.Object) []string {
		refs := ExtractDependenciesForObject(raw)
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.Kind == ConfigMap {
				out = append(out, ref.Namespace+"/"+ref.Name)
			}
		}
		return out
	}

	for _, obj := range []client.Object{&appsv1.Deployment{}, &appsv1.StatefulSet{}, &appsv1.DaemonSet{}} {
		if err := indexer.IndexField(ctx, obj, SecretReferenceIndexField, indexSecret); err != nil {
			return err
		}
		if err := indexer.IndexField(ctx, obj, ConfigMapReferenceIndexField, indexConfigMap); err != nil {
			return err
		}
	}

	return nil
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

// LookupConsumingWorkloads returns all supported workload kinds consuming a
// given source reference.
func LookupConsumingWorkloads(ctx context.Context, c client.Client, ref DependencyRef) ([]ConsumerTarget, error) {
	field := ""
	switch ref.Kind {
	case Secret:
		field = SecretReferenceIndexField
	case ConfigMap:
		field = ConfigMapReferenceIndexField
	default:
		return nil, nil
	}

	var deployments appsv1.DeploymentList
	if err := c.List(ctx, &deployments, client.InNamespace(ref.Namespace), client.MatchingFields{field: ref.Namespace + "/" + ref.Name}); err != nil {
		return nil, err
	}
	var statefulSets appsv1.StatefulSetList
	if err := c.List(ctx, &statefulSets, client.InNamespace(ref.Namespace), client.MatchingFields{field: ref.Namespace + "/" + ref.Name}); err != nil {
		return nil, err
	}
	var daemonSets appsv1.DaemonSetList
	if err := c.List(ctx, &daemonSets, client.InNamespace(ref.Namespace), client.MatchingFields{field: ref.Namespace + "/" + ref.Name}); err != nil {
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
