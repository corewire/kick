package observation

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObserveSecretMetadataOnlyAndRelevantChanges(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	observer := NewObserver(store, nil)

	baseTime := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "1", Labels: map[string]string{"a": "1"}},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("alpha")},
	}

	first, err := observer.ObserveSecret(ctx, nil, secret, baseTime)
	if err != nil {
		t.Fatalf("observe baseline: %v", err)
	}
	if first.Kind != BaselineEstablished {
		t.Fatalf("baseline kind = %s, want %s", first.Kind, BaselineEstablished)
	}

	metaOnly := secret.DeepCopy()
	metaOnly.ResourceVersion = "2"
	metaOnly.Labels["a"] = "2"
	meta, err := observer.ObserveSecret(ctx, secret, metaOnly, baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("observe metadata-only: %v", err)
	}
	if meta.Kind != MetadataOnlyChange {
		t.Fatalf("metadata kind = %s, want %s", meta.Kind, MetadataOnlyChange)
	}

	relevant := metaOnly.DeepCopy()
	relevant.ResourceVersion = "3"
	relevant.Data["token"] = []byte("beta")
	changed, err := observer.ObserveSecret(ctx, metaOnly, relevant, baseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("observe relevant: %v", err)
	}
	if changed.Kind != RelevantChange {
		t.Fatalf("relevant kind = %s, want %s", changed.Kind, RelevantChange)
	}

	identical := relevant.DeepCopy()
	identical.ResourceVersion = "3"
	none, err := observer.ObserveSecret(ctx, relevant, identical, baseTime.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("observe no change: %v", err)
	}
	if none.Kind != NoChange {
		t.Fatalf("no change kind = %s, want %s", none.Kind, NoChange)
	}
}

func TestObserveConfigMapRestartRecoveryPrototype(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	firstObserver := NewObserver(store, nil)
	baseTime := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "ns", ResourceVersion: "1"},
		Data:       map[string]string{"mode": "safe"},
	}
	if _, err := firstObserver.ObserveConfigMap(ctx, nil, cm, baseTime); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	// Simulate controller restart by creating a new observer over persisted store.
	secondObserver := NewObserver(store, nil)
	metaOnly := cm.DeepCopy()
	metaOnly.ResourceVersion = "2"
	metaOnly.Annotations = map[string]string{"x": "y"}
	meta, err := secondObserver.ObserveConfigMap(ctx, cm, metaOnly, baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("metadata-only after restart: %v", err)
	}
	if meta.Kind != MetadataOnlyChange {
		t.Fatalf("kind after restart = %s, want %s", meta.Kind, MetadataOnlyChange)
	}

	relevant := metaOnly.DeepCopy()
	relevant.ResourceVersion = "3"
	relevant.Data["mode"] = "fast"
	changed, err := secondObserver.ObserveConfigMap(ctx, metaOnly, relevant, baseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("relevant after restart: %v", err)
	}
	if changed.Kind != RelevantChange {
		t.Fatalf("kind after data change = %s, want %s", changed.Kind, RelevantChange)
	}
}
