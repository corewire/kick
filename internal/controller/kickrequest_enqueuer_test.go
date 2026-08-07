package controller

import (
	"context"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type staticPolicyMatcher struct {
	result policy.MatchResult
}

func (s staticPolicyMatcher) MatchDeployment(context.Context, *appsv1.Deployment) (policy.MatchResult, error) {
	return s.result, nil
}

func (s staticPolicyMatcher) MatchWorkload(context.Context, string, map[string]string) (policy.MatchResult, error) {
	return s.result, nil
}

func TestKickRequestEnqueuerSkipsUnmanagedWorkload(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kickv1alpha1.AddToScheme(scheme)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10"}}}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&kickv1alpha1.KickRequest{}).WithObjects(dep).Build()
	enq := &KickRequestEnqueuer{
		Client:        c,
		Coalescer:     kickrequest.NewCoalescer(c, kickrequest.RetentionConfig{}),
		PolicyMatcher: staticPolicyMatcher{result: policy.MatchResult{Managed: false, Reason: policy.ReasonPolicyDeleted}},
	}

	if err := enq.EnqueueForConsumers(context.Background(), observation.SourceIdentity{}, []dependency.ConsumerTarget{{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "team-a", Name: "api"}}, time.Now().UTC()); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	var list kickv1alpha1.KickRequestList
	if err := c.List(context.Background(), &list); err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("expected no kickrequest for unmanaged workload, got %d", len(list.Items))
	}
}
