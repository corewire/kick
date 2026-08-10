package kickrequest

import (
	"context"
	"strings"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/telemetry"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RetentionConfig is a skeleton for terminal request retention policy.
type RetentionConfig struct {
	TerminalTTL time.Duration
}

// Coalescer ensures repeated source events map to one active request per target.
type Coalescer struct {
	client.Client
	Retention RetentionConfig
	now       func() time.Time
}

func NewCoalescer(c client.Client, retention RetentionConfig) *Coalescer {
	return &Coalescer{Client: c, Retention: retention, now: time.Now}
}

// EnsureActiveRequest creates or updates one active KickRequest for a target.
func (c *Coalescer) EnsureActiveRequest(ctx context.Context, namespace string, target kickv1alpha1.ObjectReference, policyName string, latestObservedChange time.Time) (*kickv1alpha1.KickRequest, error) {
	key := types.NamespacedName{Namespace: namespace, Name: requestNameForTarget(target)}
	var request kickv1alpha1.KickRequest
	if err := c.Get(ctx, key, &request); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		request = kickv1alpha1.KickRequest{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: namespace},
			Spec:       kickv1alpha1.KickRequestSpec{TargetRef: target},
		}
		if tp := telemetry.Traceparent(ctx); tp != "" {
			request.Annotations = map[string]string{telemetry.TraceparentAnnotation: tp}
		}
		if policyName != "" {
			request.Spec.PolicyRef = &kickv1alpha1.PolicyReference{Name: policyName}
		}
		if err := c.Create(ctx, &request); err != nil {
			return nil, err
		}
	} else if request.Status.Phase == "" || isTerminalPhase(request.Status.Phase) {
		// A new restart cycle begins: re-root the trace on the change that
		// triggered it. Repeated events during an active cycle keep the
		// original traceparent so the in-flight restart stays correlated.
		if tp := telemetry.Traceparent(ctx); tp != "" && request.Annotations[telemetry.TraceparentAnnotation] != tp {
			if request.Annotations == nil {
				request.Annotations = map[string]string{}
			}
			request.Annotations[telemetry.TraceparentAnnotation] = tp
			if err := c.Update(ctx, &request); err != nil {
				return nil, err
			}
		}
	}

	if err := c.updateStatusWithRetry(ctx, key, latestObservedChange); err != nil {
		return nil, err
	}

	if err := c.Get(ctx, key, &request); err != nil {
		return nil, err
	}
	return &request, nil
}

func (c *Coalescer) updateStatusWithRetry(ctx context.Context, key types.NamespacedName, latestObservedChange time.Time) error {
	if latestObservedChange.IsZero() {
		latestObservedChange = c.now().UTC()
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var request kickv1alpha1.KickRequest
		if err := c.Get(ctx, key, &request); err != nil {
			return err
		}

		if request.Status.Phase == "" || isTerminalPhase(request.Status.Phase) {
			request.Status.Phase = kickv1alpha1.KickRequestPhasePending
			// Clear the prior rollout so the executor issues a fresh restart for
			// this change instead of adopting the already-completed rollout.
			request.Status.CurrentRollout = kickv1alpha1.RolloutStatus{}
		}

		if request.Status.LatestObservedDependencyChange == nil || latestObservedChange.After(request.Status.LatestObservedDependencyChange.Time) {
			t := metav1.NewTime(latestObservedChange.UTC())
			request.Status.LatestObservedDependencyChange = &t
		}

		return c.Status().Update(ctx, &request)
	})
}

func isTerminalPhase(phase kickv1alpha1.KickRequestPhase) bool {
	switch phase {
	case kickv1alpha1.KickRequestPhaseSucceeded, kickv1alpha1.KickRequestPhaseNoLongerRequired, kickv1alpha1.KickRequestPhaseFailed:
		return true
	default:
		return false
	}
}

func requestNameForTarget(target kickv1alpha1.ObjectReference) string {
	return strings.ToLower(target.Kind) + "-" + target.Name
}
