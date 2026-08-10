package kargo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/corewire/kick/internal/gitops"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubArgoCD struct {
	owner    gitops.Owner
	err      error
	decision gitops.GateDecision
	calls    int
}

func (s *stubArgoCD) ResolveOwner(context.Context, client.Object) (gitops.Owner, error) {
	return s.owner, s.err
}

func (s *stubArgoCD) EvaluateGate(context.Context, gitops.Owner, time.Time) (gitops.GateDecision, error) {
	s.calls++
	return s.decision, nil
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	scheme.AddKnownTypeWithName(StageGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(StageGVK.GroupVersion().WithKind("StageList"), &unstructured.UnstructuredList{})
	promotionGVK := schema.GroupVersionKind{Group: "kargo.akuity.io", Version: "v1alpha1", Kind: "Promotion"}
	scheme.AddKnownTypeWithName(promotionGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(promotionListGVK, &unstructured.UnstructuredList{})
	appGVK := schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"}
	scheme.AddKnownTypeWithName(appGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(appGVK.GroupVersion().WithKind("ApplicationList"), &unstructured.UnstructuredList{})
	return scheme
}

func application(annotation string) *unstructured.Unstructured {
	app := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{}}}
	app.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"})
	app.SetNamespace("argocd")
	app.SetName("web")
	if annotation != "" {
		app.SetAnnotations(map[string]string{authorizedStageAnnotation: annotation})
	}
	return app
}

func stage(name string, currentPromotion string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{"status": map[string]any{}}}
	obj.SetGroupVersionKind(StageGVK)
	obj.SetNamespace("shop")
	obj.SetName(name)
	if currentPromotion != "" {
		_ = unstructured.SetNestedField(obj.Object, currentPromotion, "status", "currentPromotion", "name")
	}
	return obj
}

func promotion(name, stageName, phase string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"spec":   map[string]any{"stage": stageName},
		"status": map[string]any{"phase": phase},
	}}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "kargo.akuity.io", Version: "v1alpha1", Kind: "Promotion"})
	obj.SetNamespace("shop")
	obj.SetName(name)
	return obj
}

func workload() *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "prod"}}
}

// Kargo cannot be inferred from a workload, so auto detection must never pick it.
func TestDetectIsNeverConfident(t *testing.T) {
	p := &Provider{}
	if p.Detect(workload()).Confident {
		t.Fatal("kargo must not be auto-detected")
	}
}

func TestResolveOwnerReturnsStage(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(application("shop:prod")).Build()
	p := &Provider{Client: c, ArgoCD: &stubArgoCD{owner: gitops.Owner{Namespace: "argocd", Name: "web"}}}

	owner, err := p.ResolveOwner(context.Background(), workload())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner.Kind != "Stage" || owner.Namespace != "shop" || owner.Name != "prod" {
		t.Fatalf("unexpected owner: %+v", owner)
	}
	if owner.Project != "argocd/web" {
		t.Fatalf("expected the application reference to be carried, got %q", owner.Project)
	}
}

// Multiple authorized Stages make ownership ambiguous, which must block.
func TestResolveOwnerAmbiguousStages(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(application("shop:prod,shop:staging")).Build()
	p := &Provider{Client: c, ArgoCD: &stubArgoCD{owner: gitops.Owner{Namespace: "argocd", Name: "web"}}}

	_, err := p.ResolveOwner(context.Background(), workload())
	var resolutionErr ResolutionError
	if !errors.As(err, &resolutionErr) || resolutionErr.Reason != gitops.GateAmbiguousOwner {
		t.Fatalf("expected an ambiguous owner error, got %v", err)
	}
}

func TestResolveOwnerWithoutAnnotation(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(application("")).Build()
	p := &Provider{Client: c, ArgoCD: &stubArgoCD{owner: gitops.Owner{Namespace: "argocd", Name: "web"}}}

	_, err := p.ResolveOwner(context.Background(), workload())
	var resolutionErr ResolutionError
	if !errors.As(err, &resolutionErr) || resolutionErr.Reason != gitops.GateOwnerUnknown {
		t.Fatalf("expected an unknown owner error, got %v", err)
	}
}

func TestEvaluateGateBlocksOnCurrentPromotion(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(stage("prod", "promo-1")).Build()
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true}}
	p := &Provider{Client: c, ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitops.MayExecute(decision) || decision.Reason != gitops.GateOwnerReconciling {
		t.Fatalf("expected the gate to block, got %+v", decision)
	}
	if argocd.calls != 0 {
		t.Fatal("the argocd gate must not be consulted while a promotion runs")
	}
}

func TestEvaluateGateBlocksOnPendingPromotion(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(stage("prod", ""), promotion("promo-1", "prod", "Running")).Build()
	p := &Provider{Client: c, ArgoCD: &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true}}}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gitops.MayExecute(decision) {
		t.Fatalf("expected the gate to block, got %+v", decision)
	}
}

// A promotion for a different Stage must not block this Stage.
func TestEvaluateGateIgnoresOtherStagePromotions(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(stage("prod", ""), promotion("promo-1", "staging", "Running")).Build()
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: true, Reconciled: true, Reason: gitops.GateAllowed}}
	p := &Provider{Client: c, ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gitops.MayExecute(decision) {
		t.Fatalf("expected the gate to open, got %+v", decision)
	}
	if argocd.calls != 1 {
		t.Fatalf("expected the argocd gate to be consulted once, got %d", argocd.calls)
	}
}

func TestEvaluateGateDelegatesToArgoCDWhenSettled(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme()).
		WithObjects(stage("prod", ""), promotion("promo-1", "prod", "Succeeded")).Build()
	argocd := &stubArgoCD{decision: gitops.GateDecision{Allowed: false, Reason: gitops.GateOwnerOutOfSync}}
	p := &Provider{Client: c, ArgoCD: argocd}

	decision, err := p.EvaluateGate(context.Background(), gitops.Owner{Namespace: "shop", Name: "prod", Project: "argocd/web"}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Reason != gitops.GateOwnerOutOfSync {
		t.Fatalf("expected the argocd decision to be returned, got %+v", decision)
	}
}

func TestSingleAuthorizedStage(t *testing.T) {
	cases := []struct {
		raw     string
		wantOK  bool
		project string
		stage   string
	}{
		{raw: "shop:prod", wantOK: true, project: "shop", stage: "prod"},
		{raw: " shop:prod ", wantOK: true, project: "shop", stage: "prod"},
		{raw: "shop:prod,shop:dev", wantOK: false},
		{raw: "shop", wantOK: false},
		{raw: ":prod", wantOK: false},
		{raw: "", wantOK: false},
	}
	for _, tc := range cases {
		got, ok := singleAuthorizedStage(tc.raw)
		if ok != tc.wantOK {
			t.Fatalf("%q: expected ok=%v, got %v", tc.raw, tc.wantOK, ok)
		}
		if ok && (got.project != tc.project || got.name != tc.stage) {
			t.Fatalf("%q: unexpected result %+v", tc.raw, got)
		}
	}
}
