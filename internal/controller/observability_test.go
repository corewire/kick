package controller

import (
	"testing"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/gitops"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
)

func TestPhaseToEventStableReasons(t *testing.T) {
	tests := []struct {
		name      string
		phase     kickv1alpha1.KickRequestPhase
		reason    string
		wantType  string
		wantEvent string
	}{
		{name: "waiting gate", phase: kickv1alpha1.KickRequestPhaseWaitingForGate, wantType: corev1.EventTypeNormal, wantEvent: eventWaitingForSchedule},
		{name: "waiting app sync", phase: kickv1alpha1.KickRequestPhaseWaitingForAppSync, wantType: corev1.EventTypeNormal, wantEvent: eventWaitingForGitOpsSync},
		{name: "waiting rollout", phase: kickv1alpha1.KickRequestPhaseWaitingForRollout, wantType: corev1.EventTypeNormal, wantEvent: eventWaitingForRollout},
		{name: "owner ambiguous", phase: kickv1alpha1.KickRequestPhaseWaitingForOwner, reason: string(gitops.GateAmbiguousOwner), wantType: corev1.EventTypeWarning, wantEvent: eventOwnerAmbiguous},
		{name: "owner unknown", phase: kickv1alpha1.KickRequestPhaseWaitingForOwner, reason: string(gitops.GateOwnerUnknown), wantType: corev1.EventTypeWarning, wantEvent: eventOwnerUnknown},
		{name: "executing", phase: kickv1alpha1.KickRequestPhaseExecuting, wantType: corev1.EventTypeNormal, wantEvent: eventKickStarted},
		{name: "succeeded", phase: kickv1alpha1.KickRequestPhaseSucceeded, wantType: corev1.EventTypeNormal, wantEvent: eventKickSucceeded},
		{name: "no longer required", phase: kickv1alpha1.KickRequestPhaseNoLongerRequired, wantType: corev1.EventTypeNormal, wantEvent: eventKickNoLongerRequired},
		{name: "failed", phase: kickv1alpha1.KickRequestPhaseFailed, wantType: corev1.EventTypeWarning, wantEvent: eventKickFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotEvent := phaseToEvent(tt.phase, tt.reason)
			if gotType != tt.wantType || gotEvent != tt.wantEvent {
				t.Fatalf("phaseToEvent(%s, %s) = (%s, %s), want (%s, %s)", tt.phase, tt.reason, gotType, gotEvent, tt.wantType, tt.wantEvent)
			}
		})
	}
}

func TestObservabilityCountersBoundedLabels(t *testing.T) {
	provider := "argocd"

	beforeReq := testutil.ToFloat64(kickRequestsTotal.WithLabelValues(provider, "succeeded"))
	observeRequestResult(provider, "succeeded")
	afterReq := testutil.ToFloat64(kickRequestsTotal.WithLabelValues(provider, "succeeded"))
	if afterReq-beforeReq != 1 {
		t.Fatalf("kick_requests_total delta = %v, want 1", afterReq-beforeReq)
	}

	beforeRestart := testutil.ToFloat64(kickRestartsTotal.WithLabelValues(provider, "started"))
	observeRestartResult(provider, "started")
	afterRestart := testutil.ToFloat64(kickRestartsTotal.WithLabelValues(provider, "started"))
	if afterRestart-beforeRestart != 1 {
		t.Fatalf("kick_restarts_total delta = %v, want 1", afterRestart-beforeRestart)
	}

	beforeErr := testutil.ToFloat64(kickControllerErrorsTotal.WithLabelValues("kickrequest", "UpdateStatus"))
	observeControllerError("kickrequest", "UpdateStatus")
	afterErr := testutil.ToFloat64(kickControllerErrorsTotal.WithLabelValues("kickrequest", "UpdateStatus"))
	if afterErr-beforeErr != 1 {
		t.Fatalf("kick_controller_errors_total delta = %v, want 1", afterErr-beforeErr)
	}
}
