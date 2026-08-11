package controller

import (
	"context"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ConsumerRequestEnqueuer abstracts kick request creation/coalescing.
type ConsumerRequestEnqueuer interface {
	EnqueueForConsumers(ctx context.Context, source observation.SourceIdentity, sourceLabels map[string]string, consumers []dependency.ConsumerTarget, observedAt time.Time) error
}

// NoopConsumerRequestEnqueuer is a no-op enqueuer used as a default and in tests.
type NoopConsumerRequestEnqueuer struct{}

func (NoopConsumerRequestEnqueuer) EnqueueForConsumers(context.Context, observation.SourceIdentity, map[string]string, []dependency.ConsumerTarget, time.Time) error {
	return nil
}

type SourceObservationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Observer *observation.Observer
	Enqueuer ConsumerRequestEnqueuer
	// OptionalWorkloadKinds are CRD-backed workload kinds (for example Argo
	// Rollouts) that are only looked up when their CRD is installed.
	OptionalWorkloadKinds []dependency.WorkloadKind
}

func (r *SourceObservationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if result, err := r.reconcileSecret(ctx, req); err != nil || result {
		return ctrl.Result{}, err
	}
	if result, err := r.reconcileConfigMap(ctx, req); err != nil || result {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// enqueueDependencyChange starts the trace that answers "when did the source
// change, and when did the workload restart?". It is only entered for a
// relevant change with consumers, and the executor continues this trace via the
// traceparent stamped on each KickRequest.
func (r *SourceObservationReconciler) enqueueDependencyChange(ctx context.Context, identity observation.SourceIdentity, labels map[string]string, consumers []dependency.ConsumerTarget, observedAt time.Time) error {
	ctx, span := otel.Tracer("kick.controller").Start(ctx, "dependency.changed")
	defer span.End()
	span.SetAttributes(
		attribute.String("kick.source.kind", string(identity.Kind)),
		attribute.String("kick.source.namespace", identity.Namespace),
		attribute.String("kick.source.name", identity.Name),
		attribute.Int("kick.consumers", len(consumers)),
	)
	span.AddEvent("source.changed", trace.WithTimestamp(observedAt))
	if err := r.Enqueuer.EnqueueForConsumers(ctx, identity, labels, consumers, observedAt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "enqueue failed")
		return err
	}
	return nil
}

func (r *SourceObservationReconciler) reconcileSecret(ctx context.Context, req ctrl.Request) (bool, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		return false, client.IgnoreNotFound(err)
	}

	observedAt := time.Now().UTC()
	result, err := r.Observer.ObserveSecret(ctx, nil, &secret, observedAt)
	if err != nil {
		return true, err
	}
	if result.Kind != observation.RelevantChange && result.Kind != observation.BaselineEstablished {
		return true, r.Observer.Commit(ctx, result)
	}
	consumers, err := dependency.LookupConsumingWorkloads(ctx, r.Client, dependency.DependencyRef{
		APIVersion: "v1",
		Kind:       dependency.Secret,
		Namespace:  secret.Namespace,
		Name:       secret.Name,
	}, r.OptionalWorkloadKinds...)
	if err != nil {
		return true, err
	}
	if len(consumers) == 0 {
		return true, r.Observer.Commit(ctx, result)
	}
	// Baseline and relevant changes both enqueue; baseline only reaches here when
	// consumers already reference a previously missing optional source.
	if err := r.enqueueDependencyChange(ctx, result.Identity, secret.GetLabels(), consumers, result.ChangeTime()); err != nil {
		return true, err
	}
	return true, r.Observer.Commit(ctx, result)
}

func (r *SourceObservationReconciler) reconcileConfigMap(ctx context.Context, req ctrl.Request) (bool, error) {
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &configMap); err != nil {
		return false, client.IgnoreNotFound(err)
	}

	observedAt := time.Now().UTC()
	result, err := r.Observer.ObserveConfigMap(ctx, nil, &configMap, observedAt)
	if err != nil {
		return true, err
	}
	if result.Kind != observation.RelevantChange && result.Kind != observation.BaselineEstablished {
		return true, r.Observer.Commit(ctx, result)
	}
	consumers, err := dependency.LookupConsumingWorkloads(ctx, r.Client, dependency.DependencyRef{
		APIVersion: "v1",
		Kind:       dependency.ConfigMap,
		Namespace:  configMap.Namespace,
		Name:       configMap.Name,
	}, r.OptionalWorkloadKinds...)
	if err != nil {
		return true, err
	}
	if len(consumers) == 0 {
		return true, r.Observer.Commit(ctx, result)
	}
	if err := r.enqueueDependencyChange(ctx, result.Identity, configMap.GetLabels(), consumers, result.ChangeTime()); err != nil {
		return true, err
	}
	return true, r.Observer.Commit(ctx, result)
}

func (r *SourceObservationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}}}
		})).
		Complete(r)
}
