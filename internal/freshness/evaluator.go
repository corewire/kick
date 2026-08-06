package freshness

import (
	"context"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/rollout"
	appsv1 "k8s.io/api/apps/v1"
)

// FreshnessDecision is the provider-independent stale/fresh result.
type FreshnessDecision struct {
	RestartRequired bool
	LatestChange    *time.Time
	RolloutStarted  time.Time
	BlockingReason  string
}

// Evaluator compares current dependency observations against rollout start time.
type Evaluator struct {
	Inspector rollout.RolloutInspector
}

func (e *Evaluator) Evaluate(
	ctx context.Context,
	deployment *appsv1.Deployment,
	currentDependencies []dependency.DependencyRef,
	latestRelevantChanges map[dependency.DependencyRef]time.Time,
) (FreshnessDecision, error) {
	state, err := e.Inspector.Inspect(ctx, deployment)
	if err != nil {
		return FreshnessDecision{}, err
	}

	decision := FreshnessDecision{
		RolloutStarted: state.StartedAt,
		BlockingReason: state.Reason,
	}

	if state.InProgress || state.CurrentReplicaSet.Name == "" {
		return decision, nil
	}

	var latest *time.Time
	for _, dep := range currentDependencies {
		changeAt, ok := latestRelevantChanges[dep]
		if !ok {
			continue
		}
		if latest == nil || changeAt.After(*latest) {
			copy := changeAt
			latest = &copy
		}
	}

	decision.LatestChange = latest
	if latest == nil {
		decision.RestartRequired = false
		return decision, nil
	}

	decision.RestartRequired = latest.After(state.StartedAt)
	return decision, nil
}
