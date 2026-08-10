package executor

import (
	"encoding/json"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Patching a Rollout's pod template would be read as a new revision and would
// run the full canary strategy, so restarts must use spec.restartAt.
func TestRestartPatchUsesRestartAtForArgoRollouts(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	raw, err := restartPatch(kickv1alpha1.ObjectReference{APIVersion: "argoproj.io/v1alpha1", Kind: "Rollout", Name: "web"}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	patch := map[string]any{}
	if err := json.Unmarshal(raw, &patch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	spec, ok := patch["spec"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected patch: %s", raw)
	}
	if spec["restartAt"] != now.Format(time.RFC3339) {
		t.Fatalf("unexpected restartAt: %s", raw)
	}
	if _, present := spec["template"]; present {
		t.Fatalf("a rollout patch must not touch the pod template: %s", raw)
	}
}

func TestRestartPatchUsesAnnotationForBuiltinWorkloads(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet"} {
		raw, err := restartPatch(kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: kind, Name: "web"}, now)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", kind, err)
		}
		patch := map[string]any{}
		if err := json.Unmarshal(raw, &patch); err != nil {
			t.Fatalf("%s: unexpected error: %v", kind, err)
		}
		spec := patch["spec"].(map[string]any)
		template := spec["template"].(map[string]any)
		metadata := template["metadata"].(map[string]any)
		annotations := metadata["annotations"].(map[string]any)
		if annotations[restartedAtAnnotation] != now.Format(time.RFC3339) {
			t.Fatalf("%s: unexpected patch: %s", kind, raw)
		}
	}
}

func TestRolloutCompleteHandlesArgoRollout(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Rollout",
		"spec":       map[string]any{"replicas": int64(1)},
		"status": map[string]any{
			"phase":             "Healthy",
			"updatedReplicas":   int64(1),
			"availableReplicas": int64(1),
		},
	}}
	if !rolloutComplete(obj) {
		t.Fatal("expected a healthy rollout to be complete")
	}

	_ = unstructured.SetNestedField(obj.Object, "Progressing", "status", "phase")
	if rolloutComplete(obj) {
		t.Fatal("expected a progressing rollout to be incomplete")
	}
}
