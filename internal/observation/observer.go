package observation

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ResultKind classifies observation outcomes.
type ResultKind string

const (
	BaselineEstablished ResultKind = "BaselineEstablished"
	RelevantChange      ResultKind = "RelevantChange"
	MetadataOnlyChange  ResultKind = "MetadataOnlyChange"
	NoChange            ResultKind = "NoChange"
)

// BaselinePolicy isolates unresolved initial baseline semantics.
type BaselinePolicy interface {
	OnFirstObservation(SourceIdentity) ResultKind
}

// ConservativeBaselinePolicy establishes baseline on first observation.
type ConservativeBaselinePolicy struct{}

func (ConservativeBaselinePolicy) OnFirstObservation(_ SourceIdentity) ResultKind {
	return BaselineEstablished
}

// ObservationResult is returned by observer operations.
type ObservationResult struct {
	Kind       ResultKind
	Identity   SourceIdentity
	ObservedAt time.Time
}

// Observer persists enough state to distinguish relevant and metadata-only changes.
type Observer struct {
	store          Store
	now            func() time.Time
	baselinePolicy BaselinePolicy
}

func NewObserver(store Store, baselinePolicy BaselinePolicy) *Observer {
	if baselinePolicy == nil {
		baselinePolicy = ConservativeBaselinePolicy{}
	}
	return &Observer{store: store, now: time.Now, baselinePolicy: baselinePolicy}
}

func (o *Observer) ObserveSecret(ctx context.Context, _ *corev1.Secret, newObj *corev1.Secret, observedAt time.Time) (ObservationResult, error) {
	if newObj == nil {
		return ObservationResult{Kind: NoChange}, nil
	}
	identity := SourceIdentity{APIVersion: "v1", Kind: SourceKindSecret, Namespace: newObj.Namespace, Name: newObj.Name}
	return o.observe(ctx, identity, newObj.ResourceVersion, secretFingerprint(newObj), observedAt, newObj.CreationTimestamp.Time)
}

func (o *Observer) ObserveConfigMap(ctx context.Context, _ *corev1.ConfigMap, newObj *corev1.ConfigMap, observedAt time.Time) (ObservationResult, error) {
	if newObj == nil {
		return ObservationResult{Kind: NoChange}, nil
	}
	identity := SourceIdentity{APIVersion: "v1", Kind: SourceKindConfigMap, Namespace: newObj.Namespace, Name: newObj.Name}
	return o.observe(ctx, identity, newObj.ResourceVersion, configMapFingerprint(newObj), observedAt, newObj.CreationTimestamp.Time)
}

func (o *Observer) observe(ctx context.Context, identity SourceIdentity, rv, fingerprint string, observedAt, createdAt time.Time) (ObservationResult, error) {
	if observedAt.IsZero() {
		observedAt = o.now().UTC()
	}

	record, found, err := o.store.Get(ctx, identity)
	if err != nil {
		return ObservationResult{}, err
	}
	if !found {
		kind := o.baselinePolicy.OnFirstObservation(identity)
		// A first observation establishes a baseline; anchor it to the source's
		// own creation time (deterministic) rather than the wall-clock moment KICK
		// happened to see it. A dependency created alongside its workload is then
		// never "newer" than the workload's rollout, so baselines do not trigger a
		// spurious restart, while a later-created (previously missing) source is.
		baselineTime := observedAt
		if !createdAt.IsZero() {
			baselineTime = createdAt.UTC()
		}
		record = Record{
			Identity:                    identity,
			LastSeenResourceVersion:     rv,
			LastRelevantResourceVersion: rv,
			LastRelevantChangeTime:      baselineTime,
			RelevantFingerprint:         fingerprint,
		}
		if err := o.store.Upsert(ctx, record); err != nil {
			return ObservationResult{}, err
		}
		return ObservationResult{Kind: kind, Identity: identity, ObservedAt: observedAt}, nil
	}

	if record.RelevantFingerprint == fingerprint {
		if record.LastSeenResourceVersion == rv {
			return ObservationResult{Kind: NoChange, Identity: identity, ObservedAt: observedAt}, nil
		}
		record.LastSeenResourceVersion = rv
		if err := o.store.Upsert(ctx, record); err != nil {
			return ObservationResult{}, err
		}
		return ObservationResult{Kind: MetadataOnlyChange, Identity: identity, ObservedAt: observedAt}, nil
	}

	record.LastSeenResourceVersion = rv
	record.LastRelevantResourceVersion = rv
	record.LastRelevantChangeTime = observedAt
	record.RelevantFingerprint = fingerprint
	if err := o.store.Upsert(ctx, record); err != nil {
		return ObservationResult{}, err
	}
	return ObservationResult{Kind: RelevantChange, Identity: identity, ObservedAt: observedAt}, nil
}
