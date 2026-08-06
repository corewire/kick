package dependency

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFromDeployment(t *testing.T) {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				EnvFrom: []corev1.EnvFromSource{
					{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-secret"}}},
					{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-config"}}},
				},
				Env: []corev1.EnvVar{
					{Name: "TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "key-secret"}}}},
				},
			}},
			InitContainers: []corev1.Container{{
				Name: "init",
				Env:  []corev1.EnvVar{{Name: "CFG", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "init-config"}}}}},
			}},
			Volumes: []corev1.Volume{
				{Name: "secret", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "mounted-secret"}}},
				{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "mounted-config"}}}},
				{Name: "projected", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{
					{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "projected-secret"}}},
					{ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "projected-config"}}},
				}}}},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "must-not-appear"}},
		}}},
	}

	got := FromDeployment(d)
	want := []Ref{
		{Kind: ConfigMap, Namespace: "team-a", Name: "env-config"},
		{Kind: ConfigMap, Namespace: "team-a", Name: "init-config"},
		{Kind: ConfigMap, Namespace: "team-a", Name: "mounted-config"},
		{Kind: ConfigMap, Namespace: "team-a", Name: "projected-config"},
		{Kind: Secret, Namespace: "team-a", Name: "env-secret"},
		{Kind: Secret, Namespace: "team-a", Name: "key-secret"},
		{Kind: Secret, Namespace: "team-a", Name: "mounted-secret"},
		{Kind: Secret, Namespace: "team-a", Name: "projected-secret"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d refs, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ref %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestFromDeploymentDeduplicatesReferences(t *testing.T) {
	d := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "ns"}, Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{
		{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "same"}}},
		{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "same"}}},
	}}}}}}}
	got := FromDeployment(d)
	if len(got) != 1 {
		t.Fatalf("got %#v, want one unique ref", got)
	}
}
