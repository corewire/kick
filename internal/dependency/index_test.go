package dependency

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// builderIndexer lets the production index registration run against the fake
// client builder, so the test exercises the same wiring the manager uses.
type builderIndexer struct {
	builder *fake.ClientBuilder
}

func (b builderIndexer) IndexField(_ context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	b.builder.WithIndex(obj, field, extractValue)
	return nil
}

// A controller-runtime manager does not cache unstructured objects, so a field
// index registered for a CRD-backed workload kind is never consulted and the
// field selector reaches the API server, which rejects it with "field label not
// supported". The lookup must therefore work against a client that has no index
// for the optional kind at all.
func TestLookupConsumingWorkloadsFindsArgoRolloutsWithoutAnIndex(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("scheme: %v", err)
	}
	metav1.AddToGroupVersion(scheme, ArgoRolloutGVK.GroupVersion())
	scheme.AddKnownTypeWithName(ArgoRolloutGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(ArgoRolloutGVK.GroupVersion().WithKind("RolloutList"), &unstructured.UnstructuredList{})

	consumer := newRolloutConsuming("web", "app-secret")
	other := newRolloutConsuming("api", "unrelated-secret")

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(consumer, other)
	if err := RegisterWorkloadReverseIndexes(context.Background(), builderIndexer{builder}, ArgoRolloutWorkloadKind); err != nil {
		t.Fatalf("register indexes: %v", err)
	}
	c := builder.Build()

	targets, err := LookupConsumingWorkloads(context.Background(), c, DependencyRef{
		APIVersion: "v1",
		Kind:       Secret,
		Namespace:  "default",
		Name:       "app-secret",
	}, ArgoRolloutWorkloadKind)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	want := []ConsumerTarget{{APIVersion: ArgoRolloutAPIVersion, Kind: "Rollout", Namespace: "default", Name: "web"}}
	if len(targets) != len(want) || targets[0] != want[0] {
		t.Fatalf("targets = %v, want %v", targets, want)
	}
}

func newRolloutConsuming(name, secret string) client.Object {
	obj := NewArgoRolloutObject()
	obj.SetNamespace("default")
	obj.SetName(name)
	if err := unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{
			"name": "app",
			"envFrom": []any{
				map[string]any{"secretRef": map[string]any{"name": secret}},
			},
		},
	}, "spec", "template", "spec", "containers"); err != nil {
		panic(err)
	}
	return obj
}
