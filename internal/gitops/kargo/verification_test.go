package kargo

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/gitops"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// KICK-FEAT-029: verification activity blocks; terminal outcomes do not.

func TestEvaluateGateBlocksNonTerminalVerification(t *testing.T) {
	for _, phase := range []string{"Pending", "Running", ""} {
		stage := verificationStage(phase, true, "freight-1", "fc-1", "ver-1")
		argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
		p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build(), ArgoCD: argocd}

		decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
		if err != nil {
			t.Fatalf("phase %q: %v", phase, err)
		}
		if gitops.MayExecute(decision) || decision.Reason != gitops.GateOwnerReconciling {
			t.Fatalf("phase %q: expected verification to block, got %+v", phase, decision)
		}
		if argocd.calls != 0 {
			t.Fatalf("phase %q: argocd must not be consulted while verification is active", phase)
		}
	}
}

func TestEvaluateGateBlocksUntilVerificationIsRecorded(t *testing.T) {
	stage := verificationStage("", true, "freight-1", "fc-1", "")
	stage.Object["status"].(map[string]any)["freightHistory"].([]any)[0].(map[string]any)["verificationHistory"] = []any{}
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build(), ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitops.MayExecute(decision) {
		t.Fatalf("expected missing verification record to block, got %+v", decision)
	}
	if argocd.calls != 0 {
		t.Fatal("argocd must not be consulted before verification is recorded")
	}
}

func TestEvaluateGateAllowsTerminalVerification(t *testing.T) {
	for _, phase := range []string{"Successful", "Failed", "Error", "Aborted", "Inconclusive"} {
		stage := verificationStage(phase, true, "freight-1", "fc-1", "ver-1")
		_ = unstructured.SetNestedField(stage.Object, "Unhealthy", "status", "health", "status")
		argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
		p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build(), ArgoCD: argocd}

		decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
		if err != nil {
			t.Fatalf("phase %q: %v", phase, err)
		}
		if !gitops.MayExecute(decision) {
			t.Fatalf("phase %q: terminal verification must not block, got %+v", phase, decision)
		}
	}
}

func TestEvaluateGateIgnoresOtherFreightLastPromotion(t *testing.T) {
	stage := verificationStage("", true, "freight-other", "fc-1", "")
	stage.Object["status"].(map[string]any)["freightHistory"].([]any)[0].(map[string]any)["verificationHistory"] = []any{}
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build(), ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gitops.MayExecute(decision) {
		t.Fatalf("expected other freight lastPromotion to be ignored, got %+v", decision)
	}
}

func TestEvaluateGateBlocksUnreadableFreightHistory(t *testing.T) {
	stage := stage("prod", "")
	_ = unstructured.SetNestedField(stage.Object, "not-a-list", "status", "freightHistory")
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build(), ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Reason != gitops.GateProviderUnavailable || argocd.calls != 0 {
		t.Fatalf("expected unreadable freight history to block without argocd, got %+v calls=%d", decision, argocd.calls)
	}
}

func TestRequestReverificationPatchesMatchingVerification(t *testing.T) {
	stage := verificationStage("Failed", true, "freight-1", "fc-1", "ver-1")
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()
	p := &Provider{Client: c}

	result, err := p.RequestReverification(context.Background(), recordedVerification())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if result.Status.State != kickv1alpha1.KargoReverificationRequested || result.Retry {
		t.Fatalf("unexpected result: %+v", result)
	}
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(StageGVK)
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(stage), got); err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if got.GetAnnotations()[reverifyAnnotation] != `{"id":"ver-1"}` {
		t.Fatalf("annotation = %q", got.GetAnnotations()[reverifyAnnotation])
	}
}

func TestRequestReverificationDoesNotPatchWhenAnnotationAlreadySet(t *testing.T) {
	stage := verificationStage("Failed", true, "freight-1", "fc-1", "ver-1")
	stage.SetAnnotations(map[string]string{reverifyAnnotation: `{"id":"ver-1"}`})
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()
	p := &Provider{Client: c}

	result, err := p.RequestReverification(context.Background(), recordedVerification())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if result.Status.State != kickv1alpha1.KargoReverificationRequested || result.Retry {
		t.Fatalf("expected recorded request without retry, got %+v", result)
	}
}

func TestRequestReverificationSkipsWhenFreightChanges(t *testing.T) {
	stage := verificationStage("Failed", true, "freight-1", "fc-2", "ver-1")
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()}

	result, err := p.RequestReverification(context.Background(), recordedVerification())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if result.Status.State != kickv1alpha1.KargoReverificationSkipped || stage.GetAnnotations()[reverifyAnnotation] != "" {
		t.Fatalf("expected skip without patch, got %+v annotation=%q", result, stage.GetAnnotations()[reverifyAnnotation])
	}
}

func TestRequestReverificationDoesNotPatchWhenVerificationIDChanges(t *testing.T) {
	stage := verificationStage("Failed", true, "freight-1", "fc-1", "ver-2")
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()}

	result, err := p.RequestReverification(context.Background(), recordedVerification())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if result.Retry || result.Status.State != kickv1alpha1.KargoReverificationRequested {
		t.Fatalf("expected a newer verification to count as already requested, got %+v", result)
	}
	if stage.GetAnnotations()[reverifyAnnotation] != "" {
		t.Fatal("must not patch after the verification id changes")
	}
}

func TestRequestReverificationRetriesWhileActive(t *testing.T) {
	stage := verificationStage("Running", true, "freight-1", "fc-1", "ver-1")
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()}

	result, err := p.RequestReverification(context.Background(), recordedVerification())
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if !result.Retry || result.Status.State != kickv1alpha1.KargoReverificationPending {
		t.Fatalf("expected retry without a state change, got %+v", result)
	}
}

func TestSnapshotVerificationSkipsMissingVerificationInfo(t *testing.T) {
	stage := stage("prod", "")
	p := &Provider{Client: fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage).Build()}

	status, err := p.SnapshotVerification(context.Background(), "shop", "prod")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if status.State != kickv1alpha1.KargoReverificationSkipped || status.VerificationID != "" {
		t.Fatalf("expected skip without a verification id, got %+v", status)
	}
}

func verificationStage(phase string, configured bool, lastFreight, collectionID, verificationID string) *unstructured.Unstructured {
	history := map[string]any{
		"id": collectionID,
		"items": map[string]any{
			"warehouse": map[string]any{"name": "freight-1"},
		},
	}
	if verificationID != "" || phase != "" {
		history["verificationHistory"] = []any{map[string]any{"id": verificationID, "phase": phase}}
	}
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"freightHistory": []any{history},
			"lastPromotion": map[string]any{
				"status":  map[string]any{"phase": "Succeeded"},
				"freight": map[string]any{"name": lastFreight},
			},
		},
	}}
	if configured {
		obj.Object["spec"] = map[string]any{"verification": map[string]any{"analysisTemplates": []any{map[string]any{"name": "health"}}}}
	}
	obj.SetGroupVersionKind(StageGVK)
	obj.SetNamespace("shop")
	obj.SetName("prod")
	return obj
}

func recordedVerification() kickv1alpha1.KargoReverificationStatus {
	return kickv1alpha1.KargoReverificationStatus{
		StageNamespace:      "shop",
		StageName:           "prod",
		FreightCollectionID: "fc-1",
		VerificationID:      "ver-1",
		State:               kickv1alpha1.KargoReverificationPending,
	}
}
