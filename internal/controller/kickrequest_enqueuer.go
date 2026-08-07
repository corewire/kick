package controller

import (
	"context"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KickRequestEnqueuer coalesces source events into one active request per consumer Deployment.
type KickRequestEnqueuer struct {
	Client        client.Client
	Coalescer     *kickrequest.Coalescer
	PolicyMatcher policy.WorkloadMatcher
}

func (e *KickRequestEnqueuer) EnqueueForConsumers(ctx context.Context, _ observation.SourceIdentity, sourceLabels map[string]string, consumers []dependency.ConsumerTarget, observedAt time.Time) error {
	for _, consumer := range consumers {
		targetRef := kickv1alpha1.ObjectReference{APIVersion: consumer.APIVersion, Kind: consumer.Kind, Name: consumer.Name}
		if !supportedTargetRef(targetRef) {
			continue
		}

		labels, err := consumerWorkloadLabels(ctx, e.Client, consumer)
		if err != nil {
			continue
		}

		if e.PolicyMatcher != nil {
			match, err := e.PolicyMatcher.MatchWorkload(ctx, consumer.Namespace, labels)
			if err != nil {
				return err
			}
			if !match.Managed {
				continue
			}
			// The changed dependency must be in the policy's trigger scope.
			ok, err := dependencySelectorMatches(match.Policy, sourceLabels)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
		}
		if _, err := e.Coalescer.EnsureActiveRequest(ctx, consumer.Namespace, targetRef, observedAt); err != nil {
			return err
		}
	}
	return nil
}

func consumerWorkloadLabels(ctx context.Context, c client.Client, consumer dependency.ConsumerTarget) (map[string]string, error) {
	switch consumer.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := c.Get(ctx, client.ObjectKey{Namespace: consumer.Namespace, Name: consumer.Name}, &deployment); err != nil {
			return nil, err
		}
		return deployment.GetLabels(), nil
	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		if err := c.Get(ctx, client.ObjectKey{Namespace: consumer.Namespace, Name: consumer.Name}, &statefulSet); err != nil {
			return nil, err
		}
		return statefulSet.GetLabels(), nil
	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		if err := c.Get(ctx, client.ObjectKey{Namespace: consumer.Namespace, Name: consumer.Name}, &daemonSet); err != nil {
			return nil, err
		}
		return daemonSet.GetLabels(), nil
	default:
		return nil, nil
	}
}
