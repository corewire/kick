package observation

import (
	"context"
	"sort"
	"strings"
	"time"
)

// MountedObject is one provider-reported object version taken from a
// SecretProviderClassPodStatus. It never carries secret material: the CSI driver
// only reports an object identifier and an opaque provider version.
type MountedObject struct {
	ID      string
	Version string
}

// PodMount is the set of objects one pod currently has mounted for a
// SecretProviderClass.
type PodMount struct {
	PodName string
	Objects []MountedObject
}

// SecretProviderClassFingerprint returns the deterministic fingerprint of the
// object versions a SecretProviderClass currently resolves to, plus whether the
// mounts are consistent.
//
// Pods are rotated one at a time, so mid-rotation the pods of a workload
// disagree about the current versions. Restarting on a transient disagreement
// would cause an extra rollout, so a mixed state is reported as inconsistent and
// is not observed at all; the caller retries until the pods converge.
func SecretProviderClassFingerprint(mounts []PodMount) (fingerprint string, consistent bool) {
	if len(mounts) == 0 {
		return "", false
	}

	var first string
	for i, mount := range mounts {
		current := mountFingerprintInput(mount.Objects)
		if i == 0 {
			first = current
			continue
		}
		if current != first {
			return "", false
		}
	}
	return digest(first), true
}

func mountFingerprintInput(objects []MountedObject) string {
	items := make([]string, 0, len(objects))
	for _, object := range objects {
		if object.ID == "" {
			continue
		}
		items = append(items, object.ID+"="+object.Version)
	}
	sort.Strings(items)
	return strings.Join(items, "\n")
}

// ObserveSecretProviderClass records the mounted object versions of a
// SecretProviderClass. Inconsistent mounts yield NoChange so the caller can
// retry once the rotation settles.
func (o *Observer) ObserveSecretProviderClass(ctx context.Context, namespace, name string, mounts []PodMount, observedAt time.Time) (ObservationResult, error) {
	identity := SourceIdentity{
		APIVersion: "secrets-store.csi.x-k8s.io/v1",
		Kind:       SourceKindSecretProviderClass,
		Namespace:  namespace,
		Name:       name,
	}
	fingerprint, consistent := SecretProviderClassFingerprint(mounts)
	if !consistent {
		return ObservationResult{Kind: NoChange, Identity: identity, ObservedAt: observedAt}, nil
	}
	// There is no resource version that tracks external content, so the
	// fingerprint doubles as the version: an unchanged fingerprint is NoChange.
	//
	// The baseline is anchored to the epoch instead of "now" because whatever the
	// pods have mounted right now is, by definition, what the current rollout
	// consumes. Anchoring to the observation time would make every adopted
	// workload look stale and restart once on install.
	return o.observe(ctx, identity, fingerprint, fingerprint, observedAt, epochBaseline)
}

// epochBaseline is an anchor guaranteed to predate any rollout.
var epochBaseline = time.Unix(0, 0).UTC()
