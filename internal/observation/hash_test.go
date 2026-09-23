package observation

import (
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretFingerprintIsKeyed(t *testing.T) {
	secret := &corev1.Secret{Type: corev1.SecretTypeOpaque, Data: map[string][]byte{"password": []byte("hunter2")}}
	a := newFingerprinter([]byte("key-a")).secret(secret)
	b := newFingerprinter([]byte("key-b")).secret(secret)
	if a == b {
		t.Fatal("fingerprints under different keys must differ")
	}
	if !strings.HasPrefix(a, keyedPrefix) {
		t.Fatalf("fingerprint %q lacks prefix %q", a, keyedPrefix)
	}
	if keyID(a) == "" || keyID(a) == keyID(b) {
		t.Fatalf("key IDs must be present and distinct: %q vs %q", keyID(a), keyID(b))
	}
	if again := newFingerprinter([]byte("key-a")).secret(secret); again != a {
		t.Fatal("fingerprint must be deterministic for the same key")
	}
	if sameKey(a, digest("x")) {
		t.Fatal("a keyed fingerprint must not report the same key as an unkeyed digest")
	}
}

// A key rotation (or the upgrade from the unkeyed scheme) must not be mistaken
// for a content change: an object that has not been written since it was last
// observed is re-anchored silently, one that has is treated conservatively.
func TestObserveReanchorsAfterKeyRotation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	baseTime := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "1"},
		Data:       map[string][]byte{"token": []byte("alpha")},
	}

	oldKey := NewObserver(store, nil, WithFingerprintKey([]byte("old")))
	baseline, err := oldKey.ObserveSecret(ctx, nil, secret, baseTime)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	commit(t, oldKey, baseline)

	newKey := NewObserver(store, nil, WithFingerprintKey([]byte("new")))
	unchanged, err := newKey.ObserveSecret(ctx, nil, secret, baseTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("observe after rotation: %v", err)
	}
	if unchanged.Kind != MetadataOnlyChange {
		t.Fatalf("kind after rotation with same rv = %s, want %s", unchanged.Kind, MetadataOnlyChange)
	}
	if got := unchanged.ChangeTime(); !got.Equal(baseline.ChangeTime()) {
		t.Fatalf("change time moved on rotation: %s -> %s", baseline.ChangeTime(), got)
	}
	commit(t, newKey, unchanged)

	record, _, _ := store.Get(ctx, unchanged.Identity)
	if !sameKey(record.RelevantFingerprint, newKey.fingerprinter.secret(secret)) {
		t.Fatal("record was not re-anchored under the new key")
	}

	// Identical content again under the new key is now a plain NoChange.
	none, err := newKey.ObserveSecret(ctx, nil, secret, baseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("observe after re-anchor: %v", err)
	}
	if none.Kind != NoChange {
		t.Fatalf("kind after re-anchor = %s, want %s", none.Kind, NoChange)
	}

	// Rotation while the object was also written: the write cannot be classified.
	rotatedAgain := NewObserver(store, nil, WithFingerprintKey([]byte("newer")))
	written := secret.DeepCopy()
	written.ResourceVersion = "2"
	changed, err := rotatedAgain.ObserveSecret(ctx, nil, written, baseTime.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("observe rotated+written: %v", err)
	}
	if changed.Kind != RelevantChange {
		t.Fatalf("kind for unclassifiable write = %s, want %s", changed.Kind, RelevantChange)
	}
}

// Records written before fingerprints were keyed carry a bare digest; upgrading
// must re-anchor them without a restart.
func TestObserveMigratesLegacyUnkeyedRecord(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	identity := SourceIdentity{APIVersion: "v1", Kind: SourceKindSecret, Namespace: "ns", Name: "s1"}
	legacyTime := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	if err := store.Upsert(ctx, Record{
		Identity:                    identity,
		LastSeenResourceVersion:     "7",
		LastRelevantResourceVersion: "7",
		LastRelevantChangeTime:      legacyTime,
		RelevantFingerprint:         digest("legacy"),
	}); err != nil {
		t.Fatal(err)
	}

	observer := NewObserver(store, nil, WithFingerprintKey([]byte("k")))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "7"},
		Data:       map[string][]byte{"token": []byte("alpha")},
	}
	result, err := observer.ObserveSecret(ctx, nil, secret, legacyTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != MetadataOnlyChange {
		t.Fatalf("kind = %s, want %s", result.Kind, MetadataOnlyChange)
	}
	if !result.ChangeTime().Equal(legacyTime) {
		t.Fatalf("change time = %s, want legacy %s", result.ChangeTime(), legacyTime)
	}
}
