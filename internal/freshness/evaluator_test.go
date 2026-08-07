package freshness

import (
	"context"
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/rollout"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type staticInspector struct {
	state rollout.RolloutState
	err   error
}

func (s staticInspector) Inspect(context.Context, client.Object) (rollout.RolloutState, error) {
	return s.state, s.err
}

func TestEvaluateScenarios(t *testing.T) {
	rolloutStart := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	dep := dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "ns", Name: "s1"}
	removed := dependency.DependencyRef{APIVersion: "v1", Kind: dependency.Secret, Namespace: "ns", Name: "removed"}

	tests := []struct {
		name       string
		state      rollout.RolloutState
		deps       []dependency.DependencyRef
		changes    map[dependency.DependencyRef]time.Time
		wantKick   bool
		wantBlock  string
		wantLatest *time.Time
	}{
		{
			name:  "newer dependency requires kick",
			state: rollout.RolloutState{CurrentReplicaSet: types.NamespacedName{Namespace: "ns", Name: "rs"}, StartedAt: rolloutStart, Complete: true},
			deps:  []dependency.DependencyRef{dep},
			changes: map[dependency.DependencyRef]time.Time{
				dep: rolloutStart.Add(time.Minute),
			},
			wantKick:   true,
			wantLatest: ptrTime(rolloutStart.Add(time.Minute)),
		},
		{
			name:  "newer rollout clears requirement",
			state: rollout.RolloutState{CurrentReplicaSet: types.NamespacedName{Namespace: "ns", Name: "rs"}, StartedAt: rolloutStart.Add(2 * time.Minute), Complete: true},
			deps:  []dependency.DependencyRef{dep},
			changes: map[dependency.DependencyRef]time.Time{
				dep: rolloutStart.Add(time.Minute),
			},
			wantKick:   false,
			wantLatest: ptrTime(rolloutStart.Add(time.Minute)),
		},
		{
			name:  "removed dependencies ignored",
			state: rollout.RolloutState{CurrentReplicaSet: types.NamespacedName{Namespace: "ns", Name: "rs"}, StartedAt: rolloutStart, Complete: true},
			deps:  []dependency.DependencyRef{dep},
			changes: map[dependency.DependencyRef]time.Time{
				removed: rolloutStart.Add(30 * time.Minute),
			},
			wantKick:   false,
			wantLatest: nil,
		},
		{
			name:      "active rollout blocked",
			state:     rollout.RolloutState{CurrentReplicaSet: types.NamespacedName{Namespace: "ns", Name: "rs"}, StartedAt: rolloutStart, InProgress: true, Reason: rollout.ReasonRolloutInProgress},
			deps:      []dependency.DependencyRef{dep},
			changes:   map[dependency.DependencyRef]time.Time{dep: rolloutStart.Add(time.Minute)},
			wantKick:  false,
			wantBlock: rollout.ReasonRolloutInProgress,
		},
		{
			name:  "zero replicas deterministic",
			state: rollout.RolloutState{CurrentReplicaSet: types.NamespacedName{Namespace: "ns", Name: "rs"}, StartedAt: rolloutStart, Complete: true, InProgress: false},
			deps:  []dependency.DependencyRef{dep},
			changes: map[dependency.DependencyRef]time.Time{
				dep: rolloutStart.Add(-time.Minute),
			},
			wantKick:   false,
			wantLatest: ptrTime(rolloutStart.Add(-time.Minute)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &Evaluator{Inspector: staticInspector{state: tt.state}}
			got, err := evaluator.Evaluate(context.Background(), nil, tt.deps, tt.changes)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if got.RestartRequired != tt.wantKick {
				t.Fatalf("restartRequired=%v want %v", got.RestartRequired, tt.wantKick)
			}
			if got.BlockingReason != tt.wantBlock {
				t.Fatalf("blockingReason=%q want %q", got.BlockingReason, tt.wantBlock)
			}
			if !equalTimePtr(got.LatestChange, tt.wantLatest) {
				t.Fatalf("latestChange=%v want %v", got.LatestChange, tt.wantLatest)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	copy := t
	return &copy
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
