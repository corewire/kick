package controller

import (
	"context"
	"testing"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type captureEnqueuer struct {
	calls      int
	seen       map[string]struct{}
	lastChange time.Time
}

func (e *captureEnqueuer) EnqueueForConsumers(_ context.Context, _ observation.SourceIdentity, _ map[string]string, consumers []dependency.ConsumerTarget, changedAt time.Time) error {
	e.calls++
	e.lastChange = changedAt
	if e.seen == nil {
		e.seen = map[string]struct{}{}
	}
	for _, c := range consumers {
		e.seen[c.Namespace+"/"+c.Name+"/"+c.Kind] = struct{}{}
	}
	return nil
}

// withWorkloadIndexes registers the reverse indexes the observation reconciler
// relies on, mirroring RegisterWorkloadReverseIndexes for the fake client.
func withWorkloadIndexes(b *fake.ClientBuilder) *fake.ClientBuilder {
	b = b.WithIndex(&appsv1.Deployment{}, dependency.SecretReferenceIndexField, func(raw client.Object) []string {
		return indexedRefs(raw, dependency.Secret)
	})
	b = b.WithIndex(&appsv1.Deployment{}, dependency.ConfigMapReferenceIndexField, func(raw client.Object) []string {
		return indexedRefs(raw, dependency.ConfigMap)
	})
	b = b.WithIndex(&appsv1.StatefulSet{}, dependency.SecretReferenceIndexField, func(client.Object) []string { return nil })
	b = b.WithIndex(&appsv1.StatefulSet{}, dependency.ConfigMapReferenceIndexField, func(client.Object) []string { return nil })
	b = b.WithIndex(&appsv1.DaemonSet{}, dependency.SecretReferenceIndexField, func(client.Object) []string { return nil })
	return b.WithIndex(&appsv1.DaemonSet{}, dependency.ConfigMapReferenceIndexField, func(client.Object) []string { return nil })
}

func indexedRefs(raw client.Object, kind dependency.Kind) []string {
	deps := dependency.ExtractDependenciesForObject(raw)
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		if d.Kind == kind {
			out = append(out, d.Namespace+"/"+d.Name)
		}
	}
	return out
}

func TestSourceObserverRelevantAndMetadataOnlyBehavior(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "x"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "x"}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "app",
					Image: "registry.k8s.io/pause:3.10",
					EnvFrom: []corev1.EnvFromSource{{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "s1"}},
					}},
				}}},
			},
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "1"}, Data: map[string][]byte{"k": []byte("v1")}}

	indexer := fake.NewClientBuilder().WithScheme(scheme)
	indexer = indexer.WithObjects(deployment, secret)
	c := withWorkloadIndexes(indexer).Build()

	enqueuer := &captureEnqueuer{}
	r := &SourceObservationReconciler{
		Client:   c,
		Scheme:   scheme,
		Observer: observation.NewObserver(observation.NewMemoryStore(), nil),
		Enqueuer: enqueuer,
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "s1"}})
	if err != nil {
		t.Fatalf("reconcile baseline: %v", err)
	}
	if enqueuer.calls != 1 {
		t.Fatalf("baseline enqueue calls = %d, want 1", enqueuer.calls)
	}

	var updated corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "ns", Name: "s1"}, &updated); err != nil {
		t.Fatalf("get secret before metadata update: %v", err)
	}
	updated.Labels = map[string]string{"meta": "only"}
	if err := c.Update(ctx, &updated); err != nil {
		t.Fatalf("update secret metadata: %v", err)
	}
	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "s1"}})
	if err != nil {
		t.Fatalf("reconcile metadata-only: %v", err)
	}
	if enqueuer.calls != 1 {
		t.Fatalf("metadata-only should not enqueue, calls = %d", enqueuer.calls)
	}

	var latest corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: "ns", Name: "s1"}, &latest); err != nil {
		t.Fatalf("get secret before data update: %v", err)
	}
	if latest.Data == nil {
		latest.Data = map[string][]byte{}
	}
	latest.Data["k"] = []byte("v2")
	if err := c.Update(ctx, &latest); err != nil {
		t.Fatalf("update secret data: %v", err)
	}
	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "s1"}})
	if err != nil {
		t.Fatalf("reconcile relevant change: %v", err)
	}
	if enqueuer.calls != 2 {
		t.Fatalf("relevant change should enqueue, calls = %d", enqueuer.calls)
	}
}

// A baseline is anchored to the source's last recorded write, and the request
// opened for it has to carry that same time. Enqueuing the wall-clock moment
// KICK happened to see the source would make an unchanged dependency look newer
// than the workload's rollout and trigger a restart the observation store
// itself does not justify.
func TestSourceObserverEnqueuesBaselineChangeTime(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}

	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ns"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "x"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "x"}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "app",
					Image: "registry.k8s.io/pause:3.10",
					EnvFrom: []corev1.EnvFromSource{{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "s1"}},
					}},
				}}},
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "ns", ResourceVersion: "1", CreationTimestamp: metav1.NewTime(createdAt)},
		Data:       map[string][]byte{"k": []byte("v1")},
	}

	c := withWorkloadIndexes(fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment, secret)).Build()
	enqueuer := &captureEnqueuer{}
	r := &SourceObservationReconciler{
		Client:   c,
		Scheme:   scheme,
		Observer: observation.NewObserver(observation.NewMemoryStore(), nil),
		Enqueuer: enqueuer,
	}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "s1"}}); err != nil {
		t.Fatalf("reconcile baseline: %v", err)
	}
	if enqueuer.calls != 1 {
		t.Fatalf("baseline enqueue calls = %d, want 1", enqueuer.calls)
	}
	if !enqueuer.lastChange.Equal(createdAt) {
		t.Fatalf("enqueued change time = %s, want the source's last write time %s", enqueuer.lastChange, createdAt)
	}
}
