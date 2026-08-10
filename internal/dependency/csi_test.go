package dependency

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestExtractSecretProviderClassFromCSIVolume(t *testing.T) {
	deployment := deploymentWithVolume("prod", corev1.Volume{
		Name: "secrets",
		VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{
			Driver:           SecretsStoreCSIDriver,
			VolumeAttributes: map[string]string{"secretProviderClass": "vault-db"},
		}},
	})

	refs := ExtractDependencies(deployment)

	var found bool
	for _, ref := range refs {
		if ref.Kind == SecretProviderClass && ref.Name == "vault-db" {
			found = true
			if ref.APIVersion != SecretsStoreAPIVersion {
				t.Fatalf("expected apiVersion %q, got %q", SecretsStoreAPIVersion, ref.APIVersion)
			}
			if ref.Namespace != "prod" {
				t.Fatalf("expected namespace prod, got %q", ref.Namespace)
			}
		}
	}
	if !found {
		t.Fatalf("SecretProviderClass dependency not extracted: %+v", refs)
	}
}

// Another CSI driver mounting a volume must not be mistaken for Secrets Store.
func TestExtractIgnoresForeignCSIDriver(t *testing.T) {
	deployment := deploymentWithVolume("prod", corev1.Volume{
		Name: "data",
		VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{
			Driver:           "ebs.csi.aws.com",
			VolumeAttributes: map[string]string{"secretProviderClass": "vault-db"},
		}},
	})

	for _, ref := range ExtractDependencies(deployment) {
		if ref.Kind == SecretProviderClass {
			t.Fatalf("unexpected SecretProviderClass dependency: %+v", ref)
		}
	}
}

func TestExtractDependenciesForArgoRollout(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": ArgoRolloutAPIVersion,
		"kind":       "Rollout",
		"metadata":   map[string]any{"name": "web", "namespace": "prod"},
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{},
				"spec": map[string]any{
					"containers": []any{map[string]any{
						"name":  "app",
						"image": "nginx",
						"envFrom": []any{map[string]any{
							"secretRef": map[string]any{"name": "db"},
						}},
					}},
				},
			},
		},
	}}
	obj.SetNamespace("prod")

	refs := ExtractDependenciesForObject(obj)

	var found bool
	for _, ref := range refs {
		if ref.Kind == Secret && ref.Name == "db" {
			found = true
		}
	}
	if !found {
		t.Fatalf("secret dependency not extracted from rollout template: %+v", refs)
	}
}

// A Rollout delegating to spec.workloadRef has no inline template; the
// referenced Deployment is discovered separately.
func TestExtractDependenciesForArgoRolloutWithWorkloadRef(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": ArgoRolloutAPIVersion,
		"kind":       "Rollout",
		"metadata":   map[string]any{"name": "web", "namespace": "prod"},
		"spec": map[string]any{
			"workloadRef": map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "name": "web"},
		},
	}}
	obj.SetNamespace("prod")

	if refs := ExtractDependenciesForObject(obj); len(refs) != 0 {
		t.Fatalf("expected no dependencies, got %+v", refs)
	}
}

func TestIsArgoRollout(t *testing.T) {
	if !IsArgoRollout(ArgoRolloutAPIVersion, "Rollout") {
		t.Fatal("expected argo rollout to be recognized")
	}
	if IsArgoRollout("apps/v1", "Deployment") {
		t.Fatal("deployment must not be recognized as an argo rollout")
	}
	if IsArgoRollout("other.io/v1alpha1", "Rollout") {
		t.Fatal("foreign Rollout kind must not be recognized")
	}
}

func TestNewArgoRolloutObjectHasGVK(t *testing.T) {
	obj := NewArgoRolloutObject()
	if obj.GroupVersionKind() != ArgoRolloutGVK {
		t.Fatalf("expected %v, got %v", ArgoRolloutGVK, obj.GroupVersionKind())
	}
}

func TestNewSecretProviderClassObjectHasGVK(t *testing.T) {
	obj := NewSecretProviderClassObject()
	if obj.GroupVersionKind() != SecretProviderClassGVK {
		t.Fatalf("expected %v, got %v", SecretProviderClassGVK, obj.GroupVersionKind())
	}
	if obj.GetObjectKind().GroupVersionKind().Group == "secrets-store.csi.k8s.io" {
		t.Fatal("the CSI driver name must not be used as the API group")
	}
}

func deploymentWithVolume(namespace string, volume corev1.Volume) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: namespace},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Volumes: []corev1.Volume{volume}},
			},
		},
	}
}
