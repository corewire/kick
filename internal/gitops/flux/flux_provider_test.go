package flux

import (
	"context"
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
	provider := &Provider{}
	workload := deploymentWithFluxHelmLabels("team-a", "api", "frontend", "team-a")

	owner, err := provider.ResolveOwner(context.Background(), workload)
	if err != nil {
		t.Fatalf("resolve owner: %v", err)
	}
	if owner.Kind != "HelmRelease" || owner.Name != "frontend" {
		t.Fatalf("unexpected owner: %+v", owner)
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
	conditionItems := make([]interface{}, 0, len(conditions))
	for _, c := range conditions {
		conditionItems = append(conditionItems, c)
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
		"kind":       "Kustomization",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
		"status": map[string]interface{}{
			"conditions": conditionItems,
		},
	}}
	obj.SetAPIVersion("kustomize.toolkit.fluxcd.io/v1")
	obj.SetKind("Kustomization")
	obj.SetNamespace(namespace)
	obj.SetName(name)
	return obj
}
