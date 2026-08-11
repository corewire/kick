package timeline

import (
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSourceKindCoversEveryDependencyKind(t *testing.T) {
	tests := []struct {
		name string
		kind dependency.Kind
		want observation.SourceKind
	}{
		{name: "secret", kind: dependency.Secret, want: observation.SourceKindSecret},
		{name: "configmap", kind: dependency.ConfigMap, want: observation.SourceKindConfigMap},
		{
			name: "secretproviderclass",
			kind: dependency.SecretProviderClass,
			want: observation.SourceKindSecretProviderClass,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourceKind(tc.kind); got != tc.want {
				t.Fatalf("sourceKind(%q) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestWorkloadRestartedAtReadsArgoRolloutSpec(t *testing.T) {
	restartAt := "2026-02-03T04:05:06Z"
	rollout := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": dependency.ArgoRolloutAPIVersion,
		"kind":       "Rollout",
		"metadata":   map[string]any{"name": "api", "namespace": "team-a"},
		"spec":       map[string]any{"restartAt": restartAt},
	}}

	got := workloadRestartedAt(rollout)
	if got == nil {
		t.Fatal("no restart time reported for an Argo Rollout")
	}
	want, err := time.Parse(time.RFC3339, restartAt)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("restartedAt = %s, want %s", got, want)
	}
}

func TestWorkloadRestartedAtIgnoresUnrelatedUnstructured(t *testing.T) {
	other := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"spec":       map[string]any{"restartAt": "2026-02-03T04:05:06Z"},
	}}

	if got := workloadRestartedAt(other); got != nil {
		t.Fatalf("restartedAt = %s, want nil for a non-Rollout", got)
	}
}

func TestWorkloadRestartedAtStillReadsPodTemplateAnnotation(t *testing.T) {
	deployment := &appsv1.Deployment{}
	deployment.Spec.Template.Annotations = map[string]string{
		"kubectl.kubernetes.io/restartedAt": "2026-02-03T04:05:06Z",
	}

	if got := workloadRestartedAt(deployment); got == nil {
		t.Fatal("no restart time reported for a Deployment")
	}
}
