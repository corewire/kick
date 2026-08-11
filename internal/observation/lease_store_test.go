package observation

import (
	"context"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLeaseStorePersistsAcrossObserverInstances(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := coordinationv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add lease scheme: %v", err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewLeaseStore(c)
	ctx := context.Background()

	observer1 := NewObserver(store, nil)
	at := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	res1, err := observer1.ObserveConfigMap(ctx, nil, testConfigMap("ns", "cfg", "1", map[string]string{"a": "1"}), at)
	if err != nil {
		t.Fatalf("observe baseline: %v", err)
	}
	if res1.Kind != BaselineEstablished {
		t.Fatalf("baseline kind = %s", res1.Kind)
	}
	commit(t, observer1, res1)

	observer2 := NewObserver(store, nil)
	res2, err := observer2.ObserveConfigMap(ctx, nil, testConfigMap("ns", "cfg", "2", map[string]string{"a": "1"}), at.Add(time.Minute))
	if err != nil {
		t.Fatalf("observe after restart metadata-only: %v", err)
	}
	if res2.Kind != MetadataOnlyChange {
		t.Fatalf("metadata-only kind = %s", res2.Kind)
	}
}
