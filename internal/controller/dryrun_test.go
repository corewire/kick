package controller

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	"github.com/corewire/kick/internal/notify"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type recordingDispatcher struct {
	events []notify.Event
}

func (r *recordingDispatcher) Notify(event notify.Event) {
	r.events = append(r.events, event)
}

func dryRunReconciler(t *testing.T, dispatcher notify.Dispatcher) (*KickRequestReconciler, *stubRestartExecutor) {
	t.Helper()
	scheme := testScheme(t)
	dep := testDeployment("team-a", "api")
	req := testKickRequest("team-a", "api")

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	policyObj := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			DryRun: true,
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderNone},
		},
	}
	exec := &stubRestartExecutor{}
	return &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		PolicyMatcher:      &stubPolicyMatcher{result: policy.MatchResult{Managed: true, Policy: policyObj}},
		GateResolver:       &stubGateResolver{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}},
		RestartExecutor:    exec,
		Notifier:           dispatcher,
		Clock:              func() time.Time { return time.Now().UTC() },
		RequeueInterval:    30 * time.Second,
	}, exec
}

// A dry-run policy must reach a terminal phase without ever patching a workload.
func TestKickRequestReconcileDryRunSkipsExecution(t *testing.T) {
	r, exec := dryRunReconciler(t, nil)

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if exec.calls != 0 {
		t.Fatalf("executor calls = %d, want 0", exec.calls)
	}

	var got kickv1alpha1.KickRequest
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseDryRun {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseDryRun)
	}
	if !isTerminalPhase(got.Status.Phase) {
		t.Fatal("DryRun must be a terminal phase")
	}
}

func TestKickRequestReconcileDryRunNotifies(t *testing.T) {
	dispatcher := &recordingDispatcher{}
	r, _ := dryRunReconciler(t, dispatcher)

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var found bool
	for _, event := range dispatcher.events {
		if event.Phase == string(kickv1alpha1.KickRequestPhaseDryRun) {
			found = true
			if event.TargetName != "api" || event.TargetKind != "Deployment" {
				t.Fatalf("unexpected event target: %+v", event)
			}
		}
	}
	if !found {
		t.Fatalf("expected a DryRun notification, got %+v", dispatcher.events)
	}
}

func TestPolicyDryRun(t *testing.T) {
	if policyDryRun(nil) {
		t.Fatal("a nil policy must not be dry-run")
	}
	if policyDryRun(&kickv1alpha1.KickPolicy{}) {
		t.Fatal("dry-run must default to false")
	}
	if !policyDryRun(&kickv1alpha1.KickPolicy{Spec: kickv1alpha1.KickPolicySpec{DryRun: true}}) {
		t.Fatal("expected dry-run to be enabled")
	}
}

func TestGateProviderName(t *testing.T) {
	cases := []struct {
		provider kickv1alpha1.KickPolicyProvider
		want     string
	}{
		{provider: kickv1alpha1.KickPolicyProviderArgoCD, want: "argocd"},
		{provider: kickv1alpha1.KickPolicyProviderFlux, want: "flux"},
		{provider: kickv1alpha1.KickPolicyProviderKargo, want: "kargo"},
		{provider: kickv1alpha1.KickPolicyProviderAuto, want: ""},
		{provider: kickv1alpha1.KickPolicyProviderNone, want: ""},
	}
	for _, tc := range cases {
		pol := &kickv1alpha1.KickPolicy{Spec: kickv1alpha1.KickPolicySpec{
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: tc.provider},
		}}
		if got := gateProviderName(pol); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.provider, tc.want, got)
		}
	}
	if got := gateProviderName(nil); got != "" {
		t.Fatalf("expected an empty name for a nil policy, got %q", got)
	}
}

func TestSupportedTargetRefAcceptsArgoRollout(t *testing.T) {
	if !supportedTargetRef(kickv1alpha1.ObjectReference{APIVersion: "argoproj.io/v1alpha1", Kind: "Rollout", Name: "web"}) {
		t.Fatal("expected an argo rollout target to be supported")
	}
	if supportedTargetRef(kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Rollout", Name: "web"}) {
		t.Fatal("Rollout under apps/v1 must not be supported")
	}
	if supportedTargetRef(kickv1alpha1.ObjectReference{APIVersion: "argoproj.io/v1alpha1", Kind: "Rollout"}) {
		t.Fatal("an empty name must not be supported")
	}
}
