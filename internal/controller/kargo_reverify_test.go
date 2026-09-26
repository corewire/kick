package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	"github.com/corewire/kick/internal/gitops/kargo"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// KICK-FEAT-029: reverification is a follow-up, not another restart.

type stubReverifier struct {
	snapshot kickv1alpha1.KargoReverificationStatus
	snapErr  error
	result   kargo.ReverifyResult
	err      error
	requests int
	snaps    int
}

func (s *stubReverifier) SnapshotVerification(context.Context, string, string) (kickv1alpha1.KargoReverificationStatus, error) {
	s.snaps++
	return s.snapshot, s.snapErr
}

func (s *stubReverifier) RequestReverification(context.Context, kickv1alpha1.KargoReverificationStatus) (kargo.ReverifyResult, error) {
	s.requests++
	return s.result, s.err
}

func TestCaptureReverificationBeforeRestart(t *testing.T) {
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(testDeployment("team-a", "api"), testKickRequest("team-a", "api")).Build()
	reverifier := &stubReverifier{snapshot: kickv1alpha1.KargoReverificationStatus{
		StageNamespace: "shop", StageName: "prod", FreightCollectionID: "fc-1", VerificationID: "ver-1", State: kickv1alpha1.KargoReverificationPending,
	}}
	r := &KickRequestReconciler{
		Client:             c,
		PolicyMatcher:      &stubPolicyMatcher{result: policy.MatchResult{Managed: true, Policy: kargoReverifyPolicy(false)}},
		GateResolver:       &stubGateResolver{owner: kickv1alpha1.GitOpsOwnerStatus{Provider: "kargo", Namespace: "shop", Name: "prod"}, decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}},
		RestartExecutor:    &stubRestartExecutor{},
		KargoReverifier:    reverifier,
	}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reverifier.snaps != 1 || reverifier.requests != 0 {
		t.Fatalf("snapshot=%d request=%d", reverifier.snaps, reverifier.requests)
	}
	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.KargoReverification == nil || got.Status.KargoReverification.VerificationID != "ver-1" {
		t.Fatalf("snapshot not stored: %+v", got.Status.KargoReverification)
	}
}

func TestDryRunDoesNotSnapshotReverification(t *testing.T) {
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(testDeployment("team-a", "api"), testKickRequest("team-a", "api")).Build()
	reverifier := &stubReverifier{}
	exec := &stubRestartExecutor{}
	r := &KickRequestReconciler{
		Client:             c,
		PolicyMatcher:      &stubPolicyMatcher{result: policy.MatchResult{Managed: true, Policy: kargoReverifyPolicy(true)}},
		GateResolver:       &stubGateResolver{owner: kickv1alpha1.GitOpsOwnerStatus{Provider: "kargo", Namespace: "shop", Name: "prod"}, decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}},
		RestartExecutor:    exec,
		KargoReverifier:    reverifier,
	}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reverifier.snaps != 0 || reverifier.requests != 0 || exec.calls != 0 {
		t.Fatalf("dry-run patched something: snaps=%d requests=%d exec=%d", reverifier.snaps, reverifier.requests, exec.calls)
	}
}

func TestFinishReverificationDoesNotRestart(t *testing.T) {
	scheme := testScheme(t)
	request := testKickRequest("team-a", "api")
	request.Status.Phase = kickv1alpha1.KickRequestPhaseSucceeded
	request.Status.KargoReverification = &kickv1alpha1.KargoReverificationStatus{StageNamespace: "shop", StageName: "prod", VerificationID: "ver-1", State: kickv1alpha1.KargoReverificationPending}
	request.Status.Conditions = []metav1.Condition{{Type: statusConditionProgressing, Status: metav1.ConditionFalse, Reason: "Completed", LastTransitionTime: metav1.NewTime(time.Now().UTC())}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(testDeployment("team-a", "api"), request).Build()
	reverifier := &stubReverifier{result: kargo.ReverifyResult{Status: kickv1alpha1.KargoReverificationStatus{VerificationID: "ver-1", State: kickv1alpha1.KargoReverificationRequested}}}
	exec := &stubRestartExecutor{}
	r := &KickRequestReconciler{
		Client:             c,
		PolicyMatcher:      &stubPolicyMatcher{result: policy.MatchResult{Managed: true, Policy: kargoReverifyPolicy(false)}},
		GateResolver:       &stubGateResolver{},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{}},
		RestartExecutor:    exec,
		KargoReverifier:    reverifier,
		RequeueInterval:    time.Second,
	}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if exec.calls != 0 || reverifier.requests != 1 {
		t.Fatalf("exec=%d requests=%d", exec.calls, reverifier.requests)
	}
	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.KargoReverification == nil || got.Status.KargoReverification.State != kickv1alpha1.KargoReverificationRequested {
		t.Fatalf("state = %+v", got.Status.KargoReverification)
	}
}

func TestFinishReverificationPatchErrorDoesNotRestart(t *testing.T) {
	scheme := testScheme(t)
	request := testKickRequest("team-a", "api")
	request.Status.Phase = kickv1alpha1.KickRequestPhaseSucceeded
	request.Status.KargoReverification = &kickv1alpha1.KargoReverificationStatus{VerificationID: "ver-1", State: kickv1alpha1.KargoReverificationPending}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(testDeployment("team-a", "api"), request).Build()
	reverifier := &stubReverifier{err: errors.New("patch failed")}
	exec := &stubRestartExecutor{}
	r := &KickRequestReconciler{
		Client:             c,
		PolicyMatcher:      &stubPolicyMatcher{result: policy.MatchResult{Managed: true, Policy: kargoReverifyPolicy(false)}},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{},
		RestartExecutor:    exec,
		KargoReverifier:    reverifier,
	}

	_, err := r.finishReverification(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}, request)
	if err == nil {
		t.Fatal("expected patch error")
	}
	if exec.calls != 0 {
		t.Fatal("patch failure must not restart")
	}
	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.KargoReverification.State != kickv1alpha1.KargoReverificationPending {
		t.Fatalf("state changed after failed patch: %s", got.Status.KargoReverification.State)
	}
}

func kargoReverifyPolicy(dryRun bool) *kickv1alpha1.KickPolicy {
	return &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			DryRun: dryRun,
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{
				Provider:             kickv1alpha1.KickPolicyProviderKargo,
				ReverifyAfterRestart: true,
			},
		},
	}
}
