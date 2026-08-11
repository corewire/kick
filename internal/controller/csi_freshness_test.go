package controller

import (
	"context"
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
)

func TestDependencyToSourceIdentityCoversEveryKind(t *testing.T) {
	tests := []struct {
		name     string
		kind     dependency.Kind
		wantKind observation.SourceKind
		wantOK   bool
	}{
		{name: "secret", kind: dependency.Secret, wantKind: observation.SourceKindSecret, wantOK: true},
		{name: "configmap", kind: dependency.ConfigMap, wantKind: observation.SourceKindConfigMap, wantOK: true},
		{
			name:     "secretproviderclass",
			kind:     dependency.SecretProviderClass,
			wantKind: observation.SourceKindSecretProviderClass,
			wantOK:   true,
		},
		{name: "unknown", kind: dependency.Kind("Widget"), wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			identity, ok := dependencyToSourceIdentity(dependency.DependencyRef{
				Kind:      tc.kind,
				Namespace: "team-a",
				Name:      "app",
			})
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if identity.Kind != tc.wantKind {
				t.Fatalf("kind = %q, want %q", identity.Kind, tc.wantKind)
			}
		})
	}
}

// A SecretProviderClass change must reach the freshness evaluator. Dropping it
// here silently resolves the request as NoLongerRequired.
func TestLatestRelevantChangesIncludesSecretProviderClass(t *testing.T) {
	store := observation.NewMemoryStore()
	changedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)

	dep := dependency.DependencyRef{
		APIVersion: dependency.SecretsStoreAPIVersion,
		Kind:       dependency.SecretProviderClass,
		Namespace:  "team-a",
		Name:       "app-secrets",
	}

	identity, ok := dependencyToSourceIdentity(dep)
	if !ok {
		t.Fatal("SecretProviderClass has no source identity")
	}
	if err := store.Upsert(context.Background(), observation.Record{
		Identity:               identity,
		LastRelevantChangeTime: changedAt,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	reconciler := &KickRequestReconciler{ObservationStore: store}
	changes, err := reconciler.latestRelevantChanges(context.Background(), []dependency.DependencyRef{dep})
	if err != nil {
		t.Fatalf("latestRelevantChanges: %v", err)
	}

	got, found := changes[dep]
	if !found {
		t.Fatal("SecretProviderClass change was dropped from the freshness input")
	}
	if !got.Equal(changedAt) {
		t.Fatalf("change time = %s, want %s", got, changedAt)
	}
}
