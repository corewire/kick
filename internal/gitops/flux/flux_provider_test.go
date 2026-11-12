package flux

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
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFluxOwnerResolutionContract(t *testing.T) {
	scheme := testScheme(t)
	owner := fluxKustomization("team-a", "payments")
	provider := &Provider{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(owner).Build()}
	workload := deploymentWithFluxKustomizationLabels("team-a", "api", "payments", "team-a")

	gitops.RunProviderContract(t, gitops.ContractInput{Provider: provider, Workload: workload, Now: time.Now()})
}

func TestFluxGateDecisionContract(t *testing.T) {
	scheme := testScheme(t)
	owner := fluxKustomization("team-a", "payments")
	provider := &Provider{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(owner).Build()}

	decision, err := provider.EvaluateGate(context.Background(), gitops.Owner{Provider: "flux", APIVersion: "kustomize.toolkit.fluxcd.io/v1", Kind: "Kustomization", Namespace: "team-a", Name: "payments"}, time.Now())
	if err != nil {
		t.Fatalf("evaluate gate: %v", err)
	}
	if !decision.Allowed || !decision.Reconciled || decision.Reason != gitops.GateAllowed {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestFluxProviderUnavailableBehavior(t *testing.T) {
	scheme := testScheme(t)
	provider := &Provider{Client: fake.NewClientBuilder().WithScheme(scheme).Build()}

	decision, err := provider.EvaluateGate(context.Background(), gitops.Owner{Provider: "flux", APIVersion: "kustomize.toolkit.fluxcd.io/v1", Kind: "Kustomization", Namespace: "team-a", Name: "missing"}, time.Now())
	if err != nil {
		t.Fatalf("evaluate gate: %v", err)
	}
	if decision.Reason != gitops.GateProviderUnavailable {
		t.Fatalf("reason = %s, want %s", decision.Reason, gitops.GateProviderUnavailable)
	}
}

func TestResolveOwnerSupportsHelmReleaseLabels(t *testing.T) {
	scheme := testScheme(t)
	release := fluxObject("helm.toolkit.fluxcd.io/v2", "HelmRelease", "team-a", "frontend", nil)
	provider := &Provider{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(release).Build()}
	workload := deploymentWithFluxHelmLabels("team-a", "api", "frontend", "team-a")

	owner, err := provider.ResolveOwner(context.Background(), workload)
	if err != nil {
		t.Fatalf("resolve owner: %v", err)
	}
	if owner.Kind != "HelmRelease" || owner.Name != "frontend" {
		t.Fatalf("unexpected owner: %+v", owner)
	}
}

func TestResolveOwnerBlocks(t *testing.T) {
	both := deploymentWithFluxKustomizationLabels("team-a", "api", "payments", "team-a")
	both.Labels[labelHelmReleaseName] = "frontend"
	both.Labels[labelHelmReleaseNamespace] = "team-a"

	tests := []struct {
		name     string
		workload *appsv1.Deployment
		want     gitops.GateReason
	}{
		{name: "no labels", workload: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "api"}}, want: gitops.GateOwnerUnknown},
		{name: "owner not found", workload: deploymentWithFluxKustomizationLabels("team-a", "api", "missing", "team-a"), want: gitops.GateOwnerUnknown},
		{name: "kustomization and helmrelease labels", workload: both, want: gitops.GateAmbiguousOwner},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &Provider{Client: fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(fluxKustomization("team-a", "payments")).Build()}
			_, err := provider.ResolveOwner(context.Background(), tc.workload)
			var reasoned gitops.GateReasoner
			if !errors.As(err, &reasoned) || reasoned.GateReason() != tc.want {
				t.Fatalf("err = %v, want reason %s", err, tc.want)
			}
		})
	}
}

func TestEvaluateGateMatrix(t *testing.T) {
	ready := map[string]interface{}{"type": "Ready", "status": "True", "reason": "ReconciliationSucceeded"}
	tests := []struct {
		name       string
		generation int64
		observed   int64
		suspend    bool
		conditions []map[string]interface{}
		want       gitops.GateReason
		allowed    bool
	}{
		{name: "ready and current", generation: 2, observed: 2, conditions: []map[string]interface{}{ready}, want: gitops.GateAllowed, allowed: true},
		{name: "ready but stale generation", generation: 3, observed: 2, conditions: []map[string]interface{}{ready}, want: gitops.GateOwnerReconciling},
		{name: "ready unknown while progressing", generation: 2, observed: 2, conditions: []map[string]interface{}{{"type": "Ready", "status": "Unknown", "reason": "Progressing"}}, want: gitops.GateOwnerReconciling},
		{name: "ready false", generation: 2, observed: 2, conditions: []map[string]interface{}{{"type": "Ready", "status": "False", "reason": "HealthCheckFailed"}}, want: gitops.GateOwnerOutOfSync},
		{name: "suspended", generation: 2, observed: 2, suspend: true, conditions: []map[string]interface{}{ready}, want: gitops.GateOwnerOutOfSync},
		{name: "no ready condition", generation: 2, observed: 2, want: gitops.GateOwnerUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := fluxKustomizationWithConditions("team-a", "payments", tc.conditions)
			obj.SetGeneration(tc.generation)
			obj.Object["status"].(map[string]interface{})["observedGeneration"] = tc.observed
			obj.Object["spec"] = map[string]interface{}{"suspend": tc.suspend}
			provider := &Provider{Client: fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(obj).Build()}

			decision, err := provider.EvaluateGate(context.Background(), gitops.Owner{Provider: "flux", APIVersion: "kustomize.toolkit.fluxcd.io/v1", Kind: "Kustomization", Namespace: "team-a", Name: "payments"}, time.Now())
			if err != nil {
				t.Fatalf("evaluate gate: %v", err)
			}
			if decision.Reason != tc.want || gitops.MayExecute(decision) != tc.allowed {
				t.Fatalf("decision = %+v, want reason %s allowed %v", decision, tc.want, tc.allowed)
			}
		})
	}
}

func TestEvaluateGateReconciling(t *testing.T) {
	scheme := testScheme(t)
	owner := fluxKustomizationWithConditions("team-a", "payments", []map[string]interface{}{{"type": "Ready", "status": "False", "reason": "Progressing", "message": "apply in progress"}, {"type": "Reconciling", "status": "True"}})
	provider := &Provider{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(owner).Build()}

	decision, err := provider.EvaluateGate(context.Background(), gitops.Owner{Provider: "flux", APIVersion: "kustomize.toolkit.fluxcd.io/v1", Kind: "Kustomization", Namespace: "team-a", Name: "payments"}, time.Now())
	if err != nil {
		t.Fatalf("evaluate gate: %v", err)
	}
	if decision.Reason != gitops.GateOwnerReconciling || !decision.Reconciling {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	return scheme
}

func deploymentWithFluxKustomizationLabels(namespace, name, kustomizationName, kustomizationNamespace string) *appsv1.Deployment {
	labels := map[string]string{labelKustomizationName: kustomizationName, labelKustomizationNamespace: kustomizationNamespace}
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: labels}}
}

func deploymentWithFluxHelmLabels(namespace, name, releaseName, releaseNamespace string) *appsv1.Deployment {
	labels := map[string]string{labelHelmReleaseName: releaseName, labelHelmReleaseNamespace: releaseNamespace}
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: labels}}
}

func fluxKustomization(namespace, name string) *unstructured.Unstructured {
	return fluxKustomizationWithConditions(namespace, name, []map[string]interface{}{{"type": "Ready", "status": "True", "reason": "ReconciliationSucceeded", "message": "applied"}})
}

func fluxKustomizationWithConditions(namespace, name string, conditions []map[string]interface{}) *unstructured.Unstructured {
	return fluxObject("kustomize.toolkit.fluxcd.io/v1", "Kustomization", namespace, name, conditions)
}

func fluxObject(apiVersion, kind, namespace, name string, conditions []map[string]interface{}) *unstructured.Unstructured {
	conditionItems := make([]interface{}, 0, len(conditions))
	for _, c := range conditions {
		conditionItems = append(conditionItems, c)
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": conditionItems,
		},
	}}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetNamespace(namespace)
	obj.SetName(name)
	return obj
}
