package dependency

import (
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SecretReferenceIndexField    = "kick.corewire.io/secretReferences"
	ConfigMapReferenceIndexField = "kick.corewire.io/configMapReferences"
)

// RegisterDeploymentReverseIndexes installs Deployment field indexes for source
// reverse lookups. Index values are <namespace>/<name>.
func RegisterDeploymentReverseIndexes(ctx context.Context, indexer client.FieldIndexer) error {
	if err := indexer.IndexField(ctx, &appsv1.Deployment{}, SecretReferenceIndexField, func(raw client.Object) []string {
		deployment, ok := raw.(*appsv1.Deployment)
		if !ok {
			return nil
		}
		refs := ExtractDependencies(deployment)
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.Kind != Secret {
				continue
			}
			out = append(out, ref.Namespace+"/"+ref.Name)
		}
		return out
	}); err != nil {
		return err
	}

	return indexer.IndexField(ctx, &appsv1.Deployment{}, ConfigMapReferenceIndexField, func(raw client.Object) []string {
		deployment, ok := raw.(*appsv1.Deployment)
		if !ok {
			return nil
		}
		refs := ExtractDependencies(deployment)
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if ref.Kind != ConfigMap {
				continue
			}
			out = append(out, ref.Namespace+"/"+ref.Name)
		}
		return out
	})
}

// LookupConsumingDeployments returns deployment keys consuming a given source.
func LookupConsumingDeployments(ctx context.Context, c client.Client, ref DependencyRef) ([]types.NamespacedName, error) {
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
	if err := c.List(
		ctx,
		&deployments,
		client.InNamespace(ref.Namespace),
		client.MatchingFields{field: ref.Namespace + "/" + ref.Name},
	); err != nil {
		return nil, err
	}

	keys := make([]types.NamespacedName, 0, len(deployments.Items))
	for _, deployment := range deployments.Items {
		keys = append(keys, types.NamespacedName{Namespace: deployment.Namespace, Name: deployment.Name})
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Namespace != keys[j].Namespace {
			return keys[i].Namespace < keys[j].Namespace
		}
		return keys[i].Name < keys[j].Name
	})
	return keys, nil
}
