package dependency

import (
	"slices"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractDependenciesTableDriven(t *testing.T) {
	tests := []struct {
		name string
		spec corev1.PodSpec
		want []DependencyRef
	}{
		{
			name: "containers envFrom secret and configmap",
			spec: corev1.PodSpec{Containers: []corev1.Container{{
				EnvFrom: []corev1.EnvFromSource{
					{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-secret"}}},
					{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-config"}}},
				},
			}}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "env-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "env-secret"},
			},
		},
		{
			name: "containers env secretKeyRef and configMapKeyRef",
			spec: corev1.PodSpec{Containers: []corev1.Container{{
				Env: []corev1.EnvVar{
					{Name: "S", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "key-secret"}, Optional: ptrBool(true)}}},
					{Name: "C", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "key-config"}, Optional: ptrBool(true)}}},
				},
			}}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "key-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "key-secret"},
			},
		},
		{
			name: "initContainers envFrom secret and configmap",
			spec: corev1.PodSpec{InitContainers: []corev1.Container{{
				EnvFrom: []corev1.EnvFromSource{
					{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "init-env-secret"}}},
					{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "init-env-config"}}},
				},
			}}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "init-env-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "init-env-secret"},
			},
		},
		{
			name: "initContainers env secretKeyRef and configMapKeyRef",
			spec: corev1.PodSpec{InitContainers: []corev1.Container{{
				Env: []corev1.EnvVar{
					{Name: "IS", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "init-key-secret"}}}},
					{Name: "IC", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "init-key-config"}}}},
				},
			}}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "init-key-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "init-key-secret"},
			},
		},
		{
			name: "volumes secret and configmap",
			spec: corev1.PodSpec{Volumes: []corev1.Volume{
				{Name: "s", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "vol-secret", Optional: ptrBool(true)}}},
				{Name: "c", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "vol-config"}, Optional: ptrBool(true)}}},
			}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "vol-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "vol-secret"},
			},
		},
		{
			name: "volumes projected secret and configmap",
			spec: corev1.PodSpec{Volumes: []corev1.Volume{{
				Name: "p",
				VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{
					{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "proj-secret"}, Optional: ptrBool(true)}},
					{ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "proj-config"}, Optional: ptrBool(true)}},
				}}},
			}}},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "proj-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "proj-secret"},
			},
		},
		{
			name: "deduplicates and excludes imagePullSecrets with stable ordering",
			spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "z-secret"}}},
						{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "z-secret"}}},
						{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "a-config"}}},
					},
				}},
				Volumes:          []corev1.Volume{{Name: "v", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "a-secret"}}}},
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "never-include"}},
			},
			want: []DependencyRef{
				{APIVersion: coreAPIVersion, Kind: ConfigMap, Namespace: "team-a", Name: "a-config"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "a-secret"},
				{APIVersion: coreAPIVersion, Kind: Secret, Namespace: "team-a", Name: "z-secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "team-a"},
				Spec:       appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: tt.spec}},
			}
			got := ExtractDependencies(d)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("refs mismatch\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestExtractDependenciesNilDeployment(t *testing.T) {
	if got := ExtractDependencies(nil); got != nil {
		t.Fatalf("expected nil output, got %#v", got)
	}
}

func ptrBool(v bool) *bool {
	return &v
}
