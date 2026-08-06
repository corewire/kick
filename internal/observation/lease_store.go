package observation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	observationLeaseNamePrefix = "kick-observation-"

	annAPIVersion                = "kick.corewire.io/apiVersion"
	annKind                      = "kick.corewire.io/kind"
	annObjectName                = "kick.corewire.io/objectName"
	annLastSeenResourceVersion   = "kick.corewire.io/lastSeenResourceVersion"
	annLastRelevantResourceVer   = "kick.corewire.io/lastRelevantResourceVersion"
	annLastRelevantChangeTimeUTC = "kick.corewire.io/lastRelevantChangeTime"
	annRelevantFingerprint       = "kick.corewire.io/relevantFingerprint"
)

// LeaseStore persists observation records in coordination leases.
type LeaseStore struct {
	client client.Client
}

func NewLeaseStore(c client.Client) *LeaseStore {
	return &LeaseStore{client: c}
}

func (s *LeaseStore) Get(ctx context.Context, identity SourceIdentity) (Record, bool, error) {
	name := observationLeaseName(identity)
	var lease coordinationv1.Lease
	if err := s.client.Get(ctx, types.NamespacedName{Namespace: identity.Namespace, Name: name}, &lease); err != nil {
		if apierrors.IsNotFound(err) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}

	record := Record{
		Identity:                    identity,
		LastSeenResourceVersion:     lease.Annotations[annLastSeenResourceVersion],
		LastRelevantResourceVersion: lease.Annotations[annLastRelevantResourceVer],
		RelevantFingerprint:         lease.Annotations[annRelevantFingerprint],
	}
	if rawTime := lease.Annotations[annLastRelevantChangeTimeUTC]; rawTime != "" {
		t, err := time.Parse(time.RFC3339Nano, rawTime)
		if err != nil {
			return Record{}, false, fmt.Errorf("parse %s: %w", annLastRelevantChangeTimeUTC, err)
		}
		record.LastRelevantChangeTime = t
	}
	return record, true, nil
}

func (s *LeaseStore) Upsert(ctx context.Context, record Record) error {
	name := observationLeaseName(record.Identity)
	key := types.NamespacedName{Namespace: record.Identity.Namespace, Name: name}

	var lease coordinationv1.Lease
	err := s.client.Get(ctx, key, &lease)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if apierrors.IsNotFound(err) {
		lease = coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}}
	}

	if lease.Annotations == nil {
		lease.Annotations = map[string]string{}
	}
	lease.Annotations[annAPIVersion] = record.Identity.APIVersion
	lease.Annotations[annKind] = string(record.Identity.Kind)
	lease.Annotations[annObjectName] = record.Identity.Name
	lease.Annotations[annLastSeenResourceVersion] = record.LastSeenResourceVersion
	lease.Annotations[annLastRelevantResourceVer] = record.LastRelevantResourceVersion
	lease.Annotations[annLastRelevantChangeTimeUTC] = record.LastRelevantChangeTime.UTC().Format(time.RFC3339Nano)
	lease.Annotations[annRelevantFingerprint] = record.RelevantFingerprint

	if apierrors.IsNotFound(err) {
		return s.client.Create(ctx, &lease)
	}
	return s.client.Update(ctx, &lease)
}

func observationLeaseName(identity SourceIdentity) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", identity.APIVersion, identity.Kind, identity.Namespace, identity.Name)))
	return observationLeaseNamePrefix + hex.EncodeToString(hash[:])[:20]
}
