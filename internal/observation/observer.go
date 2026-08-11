package observation

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// pending is the record that classifying this observation produced. It is
	// persisted by Commit, not by the observation itself, so that a change is
	// only forgotten once it has actually been handed to the enqueuer. Storing it
	// earlier would lose the change for good on any downstream error: the retry
	// would compare the source against the already-updated baseline and classify
	// it as unchanged.
	pending *Record
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
	return o.observe(ctx, identity, newObj.ResourceVersion, secretFingerprint(newObj), observedAt, lastWriteTime(newObj))
}

func (o *Observer) ObserveConfigMap(ctx context.Context, _ *corev1.ConfigMap, newObj *corev1.ConfigMap, observedAt time.Time) (ObservationResult, error) {
	if newObj == nil {
		return ObservationResult{Kind: NoChange}, nil
	}
	identity := SourceIdentity{APIVersion: "v1", Kind: SourceKindConfigMap, Namespace: newObj.Namespace, Name: newObj.Name}
	return o.observe(ctx, identity, newObj.ResourceVersion, configMapFingerprint(newObj), observedAt, lastWriteTime(newObj))
}

// lastWriteTime dates the content that a first observation finds. KICK has to
// place the source on the same timeline as the workload's rollout without
// having witnessed the change that produced this content, and the only evidence
// Kubernetes offers is metadata: when the object was created and when its
// managers last wrote it. Server-side field management stamps every managing
// client's last write, which makes it the closest thing Kubernetes offers to
// "when did this content come to be" for objects that carry no such field.
//
// A source that has not been written since it was created is therefore dated to
// its creation, and is newer than a rollout only if it did not exist when that
// rollout happened - exactly the case a workload consuming an optional source
// has to be restarted for. A source that was written afterwards is dated to
// that write: KICK missed it (it was not installed, it was restarting, or its
// cache had not synced yet), and dating the content to the creation instead
// would hide the rotation for good, because every later observation matches the
// baseline.
func lastWriteTime(obj metav1.Object) time.Time {
	latest := obj.GetCreationTimestamp().Time
	for _, entry := range obj.GetManagedFields() {
		if entry.Time != nil && entry.Time.After(latest) {
			latest = entry.Time.Time
		}
	}
	return latest.UTC()
}

func (o *Observer) observe(ctx context.Context, identity SourceIdentity, rv, fingerprint string, observedAt, baselineAt time.Time) (ObservationResult, error) {
	if observedAt.IsZero() {
		observedAt = o.now().UTC()
	}

	record, found, err := o.store.Get(ctx, identity)
	if err != nil {
		return ObservationResult{}, err
	}
	if !found {
		kind := o.baselinePolicy.OnFirstObservation(identity)
		// A first observation establishes a baseline; anchor it to when the content
		// was last written rather than to the wall-clock moment KICK happened to see
		// it. A dependency that has not changed since it was created is then never
		// "newer" than the workload's rollout, so adopting a workload does not
		// restart it, while a source written after that rollout still does.
		baselineTime := observedAt
		if !baselineAt.IsZero() {
			baselineTime = baselineAt.UTC()
		}
		record = Record{
			Identity:                    identity,
			LastSeenResourceVersion:     rv,
			LastRelevantResourceVersion: rv,
			LastRelevantChangeTime:      baselineTime,
			RelevantFingerprint:         fingerprint,
		}
		return ObservationResult{Kind: kind, Identity: identity, ObservedAt: observedAt, pending: &record}, nil
	}

	if record.RelevantFingerprint == fingerprint {
		if record.LastSeenResourceVersion == rv {
			return ObservationResult{Kind: NoChange, Identity: identity, ObservedAt: observedAt}, nil
		}
		record.LastSeenResourceVersion = rv
		return ObservationResult{Kind: MetadataOnlyChange, Identity: identity, ObservedAt: observedAt, pending: &record}, nil
	}

	record.LastSeenResourceVersion = rv
	record.LastRelevantResourceVersion = rv
	record.LastRelevantChangeTime = observedAt
	record.RelevantFingerprint = fingerprint
	return ObservationResult{Kind: RelevantChange, Identity: identity, ObservedAt: observedAt, pending: &record}, nil
}

// ChangeTime is the moment this observation assigned to the change. It is the
// timestamp the record carries, so a request opened for the change is evaluated
// against exactly the time the observation store will report for it. For a
// baseline that is the source's last recorded write, which keeps an unchanged
// dependency from looking newer than the workload's rollout.
func (r ObservationResult) ChangeTime() time.Time {
	if r.pending == nil {
		return r.ObservedAt
	}
	return r.pending.LastRelevantChangeTime
}

// Commit persists the observation. Callers must commit on every path that does
// not return an error, and only after the change has been enqueued, so that a
// failed hand-off is retried against the old baseline instead of being dropped.
func (o *Observer) Commit(ctx context.Context, result ObservationResult) error {
	if result.pending == nil {
		return nil
	}
	return o.store.Upsert(ctx, *result.pending)
}
