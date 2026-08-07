package controller

import (
	"context"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
}

func (r *SourceObservationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := otel.Tracer("kick.controller").Start(ctx, "source-observer.reconcile")
	defer span.End()
	span.SetAttributes(attribute.String("source.namespace", req.Namespace), attribute.String("source.name", req.Name))

	if result, err := r.reconcileSecret(ctx, req); err != nil || result {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "secret reconcile failed")
		}
		return ctrl.Result{}, err
	}
	if result, err := r.reconcileConfigMap(ctx, req); err != nil || result {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "configmap reconcile failed")
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
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
		return true, nil
	}
	consumers, err := dependency.LookupConsumingWorkloads(ctx, r.Client, dependency.DependencyRef{
		APIVersion: "v1",
		Kind:       dependency.Secret,
		Namespace:  secret.Namespace,
		Name:       secret.Name,
	})
	if err != nil {
		return true, err
	}
	if len(consumers) == 0 {
		return true, nil
	}
	if result.Kind == observation.BaselineEstablished {
		// Baseline events are only actionable when consumers already reference a
		// previously missing optional source.
		return true, r.Enqueuer.EnqueueForConsumers(ctx, result.Identity, secret.GetLabels(), consumers, observedAt)
	}
	return true, r.Enqueuer.EnqueueForConsumers(ctx, result.Identity, secret.GetLabels(), consumers, observedAt)
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
		return true, nil
	}
	consumers, err := dependency.LookupConsumingWorkloads(ctx, r.Client, dependency.DependencyRef{
		APIVersion: "v1",
		Kind:       dependency.ConfigMap,
		Namespace:  configMap.Namespace,
		Name:       configMap.Name,
	})
	if err != nil {
		return true, err
	}
	if len(consumers) == 0 {
		return true, nil
	}
	if result.Kind == observation.BaselineEstablished {
		// Baseline events are only actionable when consumers already reference a
		// previously missing optional source.
		return true, r.Enqueuer.EnqueueForConsumers(ctx, result.Identity, configMap.GetLabels(), consumers, observedAt)
	}
	return true, r.Enqueuer.EnqueueForConsumers(ctx, result.Identity, configMap.GetLabels(), consumers, observedAt)
}

func (r *SourceObservationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}}}
		})).
		Complete(r)
}
