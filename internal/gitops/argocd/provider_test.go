package argocd

import (
	"context"
	"testing"
	"time"

	"github.com/corewire/kick/internal/gitops"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestParseTrackingID(t *testing.T) {
	id, err := parseTrackingID("my-app:apps/Deployment:default/my-deployment")
	if err != nil {
		t.Fatalf("parse tracking-id: %v", err)
	}
	if id.AppName != "my-app" || id.Group != "apps" || id.Kind != "Deployment" || id.Namespace != "default" || id.Name != "my-deployment" {
		t.Fatalf("unexpected parse result: %#v", id)
	}
}

func TestResolveOwnerRejectsStaleAnnotationThenFallback(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	app := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      "payments-app",
			"namespace": "team-a",
		},
		"spec": map[string]interface{}{"project": "prod"},
		"status": map[string]interface{}{
			"resources": []interface{}{map[string]interface{}{"group": "apps", "kind": "Deployment", "namespace": "payments", "name": "payments-api"}},
		},
	}}
	app.SetGroupVersionKind(ApplicationGVK)

	provider := &Provider{
		Client:                fake.NewClientBuilder().WithScheme(scheme).WithObjects(app).Build(),
		ApplicationNamespaces: []string{"team-a"},
		ControlPlaneNamespace: "argocd",
	}

	workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      "payments-api",
		Namespace: "payments",
		Annotations: map[string]string{
			trackingIDAnnotation: "payments-app:apps/Deployment:other/payments-api",
		},
	}}
	workload.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})

	owner, reason, err := provider.resolveOwnerWithReason(context.Background(), workload)
	if err != nil {
		t.Fatalf("resolve owner: %v", err)
	}
	if reason != gitops.GateAllowed {
		t.Fatalf("reason = %s", reason)
	}
	if owner.Name != "payments-app" || owner.Namespace != "team-a" {
		t.Fatalf("owner = %#v", owner)
	}
}

func TestEvaluateGateTypedBlocks(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	app := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      "payments-app",
			"namespace": "team-a",
		},
		"spec": map[string]interface{}{"project": "prod"},
		"status": map[string]interface{}{
			"sync": map[string]interface{}{"status": "OutOfSync"},
		},
	}}
	app.SetGroupVersionKind(ApplicationGVK)

	project := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "AppProject",
		"metadata":   map[string]interface{}{"name": "prod", "namespace": "argocd"},
	}}
	project.SetGroupVersionKind(appProjectGVK)

	provider := &Provider{
		Client:                fake.NewClientBuilder().WithScheme(scheme).WithObjects(app, project).Build(),
		ApplicationNamespaces: []string{"team-a"},
		ControlPlaneNamespace: "argocd",
	}

	decision, err := provider.EvaluateGate(context.Background(), gitops.Owner{Provider: "argocd", Namespace: "team-a", Name: "payments-app", Project: "prod"}, time.Now())
	if err != nil {
		t.Fatalf("evaluate gate: %v", err)
	}
	if decision.Reason != gitops.GateOwnerOutOfSync {
		t.Fatalf("reason = %s", decision.Reason)
	}
}
