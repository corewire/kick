package controller

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/executor"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubGateResolver struct {
	owner    kickv1alpha1.GitOpsOwnerStatus
	decision gitops.GateDecision
	err      error
	calls    int
}

func (s *stubGateResolver) ResolveOwnerAndGate(context.Context, client.Object, time.Time) (kickv1alpha1.GitOpsOwnerStatus, gitops.GateDecision, error) {
	s.calls++
	return s.owner, s.decision, s.err
}

type stubFreshnessEvaluator struct {
	decision freshness.FreshnessDecision
	err      error
	calls    int
}

func (s *stubFreshnessEvaluator) Evaluate(context.Context, client.Object, []dependency.DependencyRef, map[dependency.DependencyRef]time.Time) (freshness.FreshnessDecision, error) {
	s.calls++
	return s.decision, s.err
}

type stubRestartExecutor struct {
	result executor.Result
	err    error
	calls  int
}

func (s *stubRestartExecutor) Execute(context.Context, types.NamespacedName, kickv1alpha1.ObjectReference, types.NamespacedName) (executor.Result, error) {
	s.calls++
	return s.result, s.err
}

type completingRestartExecutor struct {
	client client.Client
	calls  int
}

type stubPolicyMatcher struct {
	result policy.MatchResult
	err    error
	calls  int
}

func (s *stubPolicyMatcher) MatchWorkload(context.Context, string, map[string]string) (policy.MatchResult, error) {
	s.calls++
	return s.result, s.err
}

func (e *completingRestartExecutor) Execute(ctx context.Context, requestKey types.NamespacedName, _ kickv1alpha1.ObjectReference, _ types.NamespacedName) (executor.Result, error) {
	e.calls++
	var req kickv1alpha1.KickRequest
	if err := e.client.Get(ctx, requestKey, &req); err != nil {
		return executor.Result{}, err
	}
	req.Status.Phase = kickv1alpha1.KickRequestPhaseSucceeded
	if err := e.client.Status().Update(ctx, &req); err != nil {
		return executor.Result{}, err
	}
	return executor.Result{Complete: true}, nil
}

func TestKickRequestReconcileClosedGateWaitsDurably(t *testing.T) {
	scheme := testScheme(t)
	dep := testDeployment("team-a", "api")
	req := testKickRequest("team-a", "api")

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	requeueAt := now.Add(2 * time.Minute)
	gate := &stubGateResolver{decision: gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOutsideSchedule, Message: "window closed", RequeueAt: &requeueAt}}
	fresh := &stubFreshnessEvaluator{}
	exec := &stubRestartExecutor{}

	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       gate,
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: fresh,
		RestartExecutor:    exec,
		Clock:              func() time.Time { return now },
		RequeueInterval:    30 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter < time.Minute {
		t.Fatalf("expected durable gate requeue, got %s", result.RequeueAfter)
	}
	if fresh.calls != 0 {
		t.Fatalf("freshness evaluator should not run for closed gate")
	}
	if exec.calls != 0 {
		t.Fatalf("executor should not run for closed gate")
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseWaitingForGate {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseWaitingForGate)
	}
	if got.Status.Gate.Reason != string(gitops.GateOutsideSchedule) {
		t.Fatalf("gate reason = %s", got.Status.Gate.Reason)
	}
}

func TestKickRequestReconcileOpenGateNoLongerRequired(t *testing.T) {
	scheme := testScheme(t)
	dep := testDeployment("team-a", "api")
	req := testKickRequest("team-a", "api")

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	gate := &stubGateResolver{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: "allowed"}}
	fresh := &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: false}}
	exec := &stubRestartExecutor{}

	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       gate,
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: fresh,
		RestartExecutor:    exec,
		Clock:              func() time.Time { return time.Now().UTC() },
		RequeueInterval:    30 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %s", result.RequeueAfter)
	}
	if exec.calls != 0 {
		t.Fatalf("executor should not run when restart is not required")
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseNoLongerRequired {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseNoLongerRequired)
	}
}

func TestKickRequestReconcileRestartRequiredTriggersExecutor(t *testing.T) {
	scheme := testScheme(t)
	dep := testDeployment("team-a", "api")
	req := testKickRequest("team-a", "api")

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	gate := &stubGateResolver{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: "allowed"}}
	fresh := &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}}
	exec := &stubRestartExecutor{}

	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       gate,
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: fresh,
		RestartExecutor:    exec,
		Clock:              func() time.Time { return time.Now().UTC() },
		RequeueInterval:    30 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 30*time.Second {
		t.Fatalf("requeue = %s, want %s", result.RequeueAfter, 30*time.Second)
	}
	if exec.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", exec.calls)
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseExecuting {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseExecuting)
	}
}

func TestKickRequestReconcileRecoversExecutingRequest(t *testing.T) {
	scheme := testScheme(t)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	startedAt := metav1.NewTime(now.Add(-time.Minute))

	dep := testDeployment("team-a", "api")
	replicas := int32(1)
	dep.Spec.Replicas = &replicas
	dep.Generation = 1
	dep.Status.ObservedGeneration = 1
	dep.Status.UpdatedReplicas = 1
	dep.Status.AvailableReplicas = 1
	dep.Status.Replicas = 1

	req := testKickRequest("team-a", "api")
	req.Status = kickv1alpha1.KickRequestStatus{
		Phase:          kickv1alpha1.KickRequestPhaseExecuting,
		CurrentRollout: kickv1alpha1.RolloutStatus{StartedAt: &startedAt},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	gate := &stubGateResolver{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: "allowed"}}
	fresh := &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}}
	completeExec := &completingRestartExecutor{client: c}

	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       gate,
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: fresh,
		RestartExecutor:    completeExec,
		Clock:              func() time.Time { return now },
		RequeueInterval:    30 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %s", result.RequeueAfter)
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseSucceeded {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseSucceeded)
	}
}

func TestKickRequestReconcilePolicyUnmanagedCancelsRequest(t *testing.T) {
	scheme := testScheme(t)
	dep := testDeployment("team-a", "api")
	req := testKickRequest("team-a", "api")

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep, req).Build()
	gate := &stubGateResolver{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: "allowed"}}
	fresh := &stubFreshnessEvaluator{decision: freshness.FreshnessDecision{RestartRequired: true}}
	exec := &stubRestartExecutor{}
	matcher := &stubPolicyMatcher{result: policy.MatchResult{Managed: false, Reason: policy.ReasonPolicyDeleted}}

	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		PolicyMatcher:      matcher,
		GateResolver:       gate,
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: fresh,
		RestartExecutor:    exec,
		Clock:              func() time.Time { return time.Now().UTC() },
		RequeueInterval:    30 * time.Second,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %s", result.RequeueAfter)
	}
	if gate.calls != 0 || fresh.calls != 0 || exec.calls != 0 {
		t.Fatalf("expected policy cancellation before gate/freshness/executor")
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhaseFailed {
		t.Fatalf("phase = %s, want %s", got.Status.Phase, kickv1alpha1.KickRequestPhaseFailed)
	}
}

func TestKickRequestReconcileTerminalRequeuesBeforeRetention(t *testing.T) {
	scheme := testScheme(t)
	req := testKickRequest("team-a", "api")
	req.Status.Phase = kickv1alpha1.KickRequestPhaseNoLongerRequired
	req.Status.Conditions = []metav1.Condition{{
		Type:               statusConditionProgressing,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Date(2026, 8, 7, 11, 50, 0, 0, time.UTC)),
	}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(req).Build()
	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       &stubGateResolver{},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{},
		RestartExecutor:    &stubRestartExecutor{},
		Clock:              func() time.Time { return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC) },
		RequestRetention:   30 * time.Minute,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 20*time.Minute {
		t.Fatalf("requeue = %s, want %s", result.RequeueAfter, 20*time.Minute)
	}

	var got kickv1alpha1.KickRequest
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("terminal request should still exist: %v", err)
	}
}

func TestKickRequestReconcileTerminalDeletesAfterRetention(t *testing.T) {
	scheme := testScheme(t)
	req := testKickRequest("team-a", "api")
	req.Status.Phase = kickv1alpha1.KickRequestPhaseSucceeded
	req.Status.Conditions = []metav1.Condition{{
		Type:               statusConditionProgressing,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)),
	}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(req).Build()
	r := &KickRequestReconciler{
		Client:             c,
		Scheme:             scheme,
		GateResolver:       &stubGateResolver{},
		ObservationStore:   observation.NewMemoryStore(),
		FreshnessEvaluator: &stubFreshnessEvaluator{},
		RestartExecutor:    &stubRestartExecutor{},
		Clock:              func() time.Time { return time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC) },
		RequestRetention:   30 * time.Minute,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "team-a", Name: "api"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %s", result.RequeueAfter)
	}

	var got kickv1alpha1.KickRequest
	err = c.Get(context.Background(), types.NamespacedName{Namespace: "team-a", Name: "api"}, &got)
	if err == nil {
		t.Fatalf("expected request to be deleted")
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}
	return scheme
}

func testKickRequest(namespace, name string) *kickv1alpha1.KickRequest {
	return &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: name}},
	}
}

func testDeployment(namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Annotations: map[string]string{"argocd.argoproj.io/tracking-id": "app:apps/Deployment:" + namespace + "/" + name}},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}},
			},
		},
	}
}
