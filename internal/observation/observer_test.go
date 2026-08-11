package observation

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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
	commit(t, observer, first)

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
	commit(t, observer, meta)

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
	commit(t, observer, changed)

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
	baseline, err := firstObserver.ObserveConfigMap(ctx, nil, cm, baseTime)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	commit(t, firstObserver, baseline)

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
	commit(t, secondObserver, meta)

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

// TestObserveKeepsReportingChangeUntilCommitted pins the contract the source
// controllers rely on: a change that could not be handed to the enqueuer must
// still be reported on the next observation instead of being swallowed by an
// already-advanced baseline.
func TestObserveKeepsReportingChangeUntilCommitted(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	observer := NewObserver(store, nil)

	baseTime := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "1"},
		Data:       map[string][]byte{"token": []byte("alpha")},
	}
	baseline, err := observer.ObserveSecret(ctx, nil, secret, baseTime)
	if err != nil {
		t.Fatalf("observe baseline: %v", err)
	}
	commit(t, observer, baseline)

	rotated := secret.DeepCopy()
	rotated.ResourceVersion = "2"
	rotated.Data["token"] = []byte("beta")

	// The enqueue fails, so nothing is committed.
	dropped, err := observer.ObserveSecret(ctx, secret, rotated, baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("observe rotation: %v", err)
	}
	if dropped.Kind != RelevantChange {
		t.Fatalf("first rotation kind = %s, want %s", dropped.Kind, RelevantChange)
	}

	retried, err := observer.ObserveSecret(ctx, secret, rotated, baseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("re-observe rotation: %v", err)
	}
	if retried.Kind != RelevantChange {
		t.Fatalf("retried rotation kind = %s, want %s", retried.Kind, RelevantChange)
	}
	commit(t, observer, retried)

	settled, err := observer.ObserveSecret(ctx, rotated, rotated, baseTime.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("observe after commit: %v", err)
	}
	if settled.Kind != NoChange {
		t.Fatalf("kind after commit = %s, want %s", settled.Kind, NoChange)
	}
}

func commit(t *testing.T, observer *Observer, result ObservationResult) {
	t.Helper()
	if err := observer.Commit(context.Background(), result); err != nil {
		t.Fatalf("commit observation: %v", err)
	}
}

// A first observation cannot witness the change that produced the content it
// sees, so it has to date that content from metadata. A source that was never
// written since it was created is dated to its creation; a source that carries
// a later write is evidence of a rotation KICK missed and is dated to that
// write, because dating it to the creation would place the rotation before the
// workload's rollout and it would never be acted on.
func TestObserveBaselineAnchorsToLastWrite(t *testing.T) {
	createdAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	writtenAt := createdAt.Add(10 * time.Minute)
	observedAt := createdAt.Add(time.Hour)

	tests := []struct {
		name     string
		managed  []metav1.ManagedFieldsEntry
		expected time.Time
	}{
		{name: "never written since creation", expected: createdAt},
		{
			name: "rotated before the first observation",
			managed: []metav1.ManagedFieldsEntry{
				{Manager: "gitops", Time: ptr.To(metav1.NewTime(writtenAt))},
				{Manager: "kubectl", Time: ptr.To(metav1.NewTime(createdAt))},
			},
			expected: writtenAt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserver(NewMemoryStore(), nil)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "s1",
					Namespace:         "ns",
					ResourceVersion:   "1",
					CreationTimestamp: metav1.NewTime(createdAt),
					ManagedFields:     tc.managed,
				},
				Data: map[string][]byte{"token": []byte("alpha")},
			}

			result, err := observer.ObserveSecret(context.Background(), nil, secret, observedAt)
			if err != nil {
				t.Fatalf("observe baseline: %v", err)
			}
			if result.Kind != BaselineEstablished {
				t.Fatalf("kind = %s, want %s", result.Kind, BaselineEstablished)
			}
			if !result.ChangeTime().Equal(tc.expected) {
				t.Fatalf("baseline change time = %s, want %s", result.ChangeTime(), tc.expected)
			}
		})
	}
}
