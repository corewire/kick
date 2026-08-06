package controller

import (
	"context"
	"time"

	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KickRequestEnqueuer coalesces source events into one active request per consumer Deployment.
type KickRequestEnqueuer struct {
	Client        client.Client
	Coalescer     *kickrequest.Coalescer
	PolicyMatcher policy.DeploymentMatcher
}

func (e *KickRequestEnqueuer) EnqueueForConsumers(ctx context.Context, _ observation.SourceIdentity, consumers []types.NamespacedName, observedAt time.Time) error {
	for _, consumer := range consumers {
		if e.PolicyMatcher != nil {
			var deployment appsv1.Deployment
			if err := e.Client.Get(ctx, consumer, &deployment); err != nil {
				continue
			}
			match, err := e.PolicyMatcher.MatchDeployment(ctx, &deployment)
			if err != nil {
				return err
			}
			if !match.Managed {
				continue
			}
		}
		if _, err := e.Coalescer.EnsureActiveRequest(ctx, consumer, observedAt); err != nil {
			return err
		}
	}
	return nil
}
