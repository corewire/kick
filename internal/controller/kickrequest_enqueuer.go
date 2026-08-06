package controller

import (
	"context"
	"time"

	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/observation"
	"k8s.io/apimachinery/pkg/types"
)

// KickRequestEnqueuer coalesces source events into one active request per consumer Deployment.
type KickRequestEnqueuer struct {
	Coalescer *kickrequest.Coalescer
}

func (e *KickRequestEnqueuer) EnqueueForConsumers(ctx context.Context, _ observation.SourceIdentity, consumers []types.NamespacedName, observedAt time.Time) error {
	for _, consumer := range consumers {
		if _, err := e.Coalescer.EnsureActiveRequest(ctx, consumer, observedAt); err != nil {
			return err
		}
	}
	return nil
}
