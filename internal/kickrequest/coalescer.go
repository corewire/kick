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
	if err := c.ensureRequestObject(ctx, key, target, policyName, latestObservedChange); err != nil {
		return nil, err
	}

	if err := c.updateStatusWithRetry(ctx, key, latestObservedChange); err != nil {
		return nil, err
	}

	var request kickv1alpha1.KickRequest
	if err := c.Get(ctx, key, &request); err != nil {
		return nil, err
	}
	return &request, nil
}

// ensureRequestObject creates the request when absent, otherwise re-roots its
// trace if a new restart cycle is starting.
func (c *Coalescer) ensureRequestObject(ctx context.Context, key types.NamespacedName, target kickv1alpha1.ObjectReference, policyName string, latestObservedChange time.Time) error {
	var request kickv1alpha1.KickRequest
	err := c.Get(ctx, key, &request)
	if apierrors.IsNotFound(err) {
		return c.createRequest(ctx, key, target, policyName)
	}
	if err != nil {
		return err
	}

	if request.Status.Phase != "" && !isTerminalPhase(request.Status.Phase) {
		return nil
	}
	if !startsNewCycle(&request, latestObservedChange) {
		return nil
	}
	return c.rerootTrace(ctx, &request)
}

func (c *Coalescer) createRequest(ctx context.Context, key types.NamespacedName, target kickv1alpha1.ObjectReference, policyName string) error {
	request := kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: target},
	}
	if tp := telemetry.Traceparent(ctx); tp != "" {
		request.Annotations = map[string]string{telemetry.TraceparentAnnotation: tp}
	}
	if policyName != "" {
		request.Spec.PolicyRef = &kickv1alpha1.PolicyReference{Name: policyName}
	}
	return c.Create(ctx, &request)
}

// rerootTrace re-roots a request on the change that starts a new restart cycle.
// Repeated events during an active cycle keep the original traceparent so the
// in-flight restart stays correlated.
func (c *Coalescer) rerootTrace(ctx context.Context, request *kickv1alpha1.KickRequest) error {
	tp := telemetry.Traceparent(ctx)
	if tp == "" || request.Annotations[telemetry.TraceparentAnnotation] == tp {
		return nil
	}

	if request.Annotations == nil {
		request.Annotations = map[string]string{}
	}
	request.Annotations[telemetry.TraceparentAnnotation] = tp
	return c.Update(ctx, request)
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

		// Hand-off from the observer is at-least-once: an observation is only
		// forgotten once it has been enqueued, so the same change can arrive
		// twice. Re-opening the request for a change it already recorded would
		// throw away the outcome KICK reached for exactly that change.
		if !startsNewCycle(&request, latestObservedChange) {
			return nil
		}

		if request.Status.Phase == "" || isTerminalPhase(request.Status.Phase) {
			request.Status.Phase = kickv1alpha1.KickRequestPhasePending
			// Clear the prior rollout so the executor issues a fresh restart for
			// this change instead of adopting the already-completed rollout.
			request.Status.CurrentRollout = kickv1alpha1.RolloutStatus{}
		}

		t := metav1.NewMicroTime(latestObservedChange.UTC())
		request.Status.LatestObservedDependencyChange = &t

		return c.Status().Update(ctx, &request)
	})
}

// startsNewCycle reports whether the observed change is newer than the one the
// request already carries, and therefore opens a new restart cycle.
func startsNewCycle(request *kickv1alpha1.KickRequest, latestObservedChange time.Time) bool {
	known := request.Status.LatestObservedDependencyChange
	return known == nil || latestObservedChange.After(known.Time)
}

func isTerminalPhase(phase kickv1alpha1.KickRequestPhase) bool {
	switch phase {
	case kickv1alpha1.KickRequestPhaseSucceeded, kickv1alpha1.KickRequestPhaseNoLongerRequired, kickv1alpha1.KickRequestPhaseFailed, kickv1alpha1.KickRequestPhaseDryRun:
		return true
	default:
		return false
	}
}

func requestNameForTarget(target kickv1alpha1.ObjectReference) string {
	return strings.ToLower(target.Kind) + "-" + target.Name
}
