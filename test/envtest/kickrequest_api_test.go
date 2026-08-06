package envtest

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/kickrequest"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestKickRequestDefaultsAndValidationEnvtest(t *testing.T) {
	t.Parallel()

	env := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")}}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("stop envtest: %v", err)
		}
	}()
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	valid := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec: kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "api",
		}},
	}
	if err := c.Create(ctx, valid); err != nil {
		t.Fatalf("create valid kickrequest: %v", err)
	}
	var got kickv1alpha1.KickRequest
	if err := c.Get(ctx, types.NamespacedName{Namespace: "team-a", Name: "api"}, &got); err != nil {
		t.Fatalf("get valid kickrequest: %v", err)
	}
	if got.Spec.TargetRef.APIVersion != "apps/v1" || got.Spec.TargetRef.Kind != "Deployment" || got.Spec.TargetRef.Name != "api" {
		t.Fatalf("unexpected persisted targetRef: %#v", got.Spec.TargetRef)
	}

	invalid := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "bad", Namespace: "team-a"},
		Spec: kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       "bad",
		}},
	}
	if err := c.Create(ctx, invalid); err == nil {
		t.Fatalf("expected validation error for invalid kind")
	}
}

func TestKickRequestConflictRetryEnvtest(t *testing.T) {
	t.Parallel()

	env := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")}}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("stop envtest: %v", err)
		}
	}()
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}
	realClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()
	if err := realClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-b"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	wrapped := &conflictOnceClient{Client: realClient}
	coalescer := kickrequest.NewCoalescer(wrapped, kickrequest.RetentionConfig{})
	at := time.Date(2026, 8, 6, 11, 0, 0, 0, time.UTC)
	if _, err := coalescer.EnsureActiveRequest(ctx, types.NamespacedName{Namespace: "team-b", Name: "api"}, at); err != nil {
		t.Fatalf("ensure with conflict retry: %v", err)
	}
	if !wrapped.injected {
		t.Fatalf("expected injected conflict to be exercised")
	}

	var got kickv1alpha1.KickRequest
	if err := realClient.Get(ctx, types.NamespacedName{Namespace: "team-b", Name: "api"}, &got); err != nil {
		t.Fatalf("get kickrequest: %v", err)
	}
	if got.Status.Phase != kickv1alpha1.KickRequestPhasePending {
		t.Fatalf("phase = %s, want pending", got.Status.Phase)
	}
}

type conflictOnceClient struct {
	client.Client
	injected bool
}

func (c *conflictOnceClient) Status() client.SubResourceWriter {
	return &conflictOnceStatusWriter{SubResourceWriter: c.Client.Status(), parent: c}
}

type conflictOnceStatusWriter struct {
	client.SubResourceWriter
	parent *conflictOnceClient
}

func (w *conflictOnceStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if !w.parent.injected {
		w.parent.injected = true
		gvk := obj.GetObjectKind().GroupVersionKind()
		return apierrors.NewConflict(schema.GroupResource{Group: gvk.Group, Resource: "kickrequests"}, obj.GetName(), errors.New("simulated conflict"))
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}
