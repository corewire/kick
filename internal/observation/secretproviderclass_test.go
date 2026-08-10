package observation

import (
	"testing"
)

func TestSecretProviderClassFingerprintIsOrderIndependent(t *testing.T) {
	a, ok := SecretProviderClassFingerprint([]PodMount{
		{PodName: "web-1", Objects: []MountedObject{{ID: "secret/a", Version: "1"}, {ID: "secret/b", Version: "2"}}},
	})
	if !ok {
		t.Fatal("expected a consistent fingerprint")
	}
	b, ok := SecretProviderClassFingerprint([]PodMount{
		{PodName: "web-1", Objects: []MountedObject{{ID: "secret/b", Version: "2"}, {ID: "secret/a", Version: "1"}}},
	})
	if !ok {
		t.Fatal("expected a consistent fingerprint")
	}
	if a != b {
		t.Fatalf("fingerprint depends on object order: %q vs %q", a, b)
	}
}

// Mid-rotation some pods still mount the previous version. Reporting a change
// then would restart the workload twice, so a mixed state must be suppressed.
func TestSecretProviderClassFingerprintRejectsMixedVersions(t *testing.T) {
	_, ok := SecretProviderClassFingerprint([]PodMount{
		{PodName: "web-1", Objects: []MountedObject{{ID: "secret/a", Version: "1"}}},
		{PodName: "web-2", Objects: []MountedObject{{ID: "secret/a", Version: "2"}}},
	})
	if ok {
		t.Fatal("expected an inconsistent result while pods disagree")
	}
}

func TestSecretProviderClassFingerprintAgreeingPodsAreConsistent(t *testing.T) {
	fp, ok := SecretProviderClassFingerprint([]PodMount{
		{PodName: "web-1", Objects: []MountedObject{{ID: "secret/a", Version: "2"}}},
		{PodName: "web-2", Objects: []MountedObject{{ID: "secret/a", Version: "2"}}},
	})
	if !ok {
		t.Fatal("expected a consistent fingerprint")
	}
	if fp == "" {
		t.Fatal("expected a non-empty fingerprint")
	}
}

func TestSecretProviderClassFingerprintNoMounts(t *testing.T) {
	if _, ok := SecretProviderClassFingerprint(nil); ok {
		t.Fatal("expected no fingerprint without mounts")
	}
}
