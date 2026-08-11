package envtest

import (
	"context"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/controller"
)

// A NotificationPolicy must report whether it can deliver before it is ever
// asked to, so a missing credential is visible on a quiet namespace.
func TestNotificationPolicyReadinessEnvtest(t *testing.T) {
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
	const namespace = "notification-readiness"
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	policy := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "alerts", Namespace: namespace},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{
				URL: "https://alerts.example.com/hooks/kick",
				Auth: &kickv1alpha1.NotificationAuth{
					BearerToken: &kickv1alpha1.SecretKeyRef{Name: "webhook-credentials", Key: "token"},
				},
			},
		},
	}
	if err := c.Create(ctx, policy); err != nil {
		t.Fatalf("create policy: %v", err)
	}

	reconciler := &controller.NotificationPolicyReconciler{Client: c, Scheme: scheme}
	key := types.NamespacedName{Namespace: namespace, Name: "alerts"}
	request := ctrl.Request{NamespacedName: key}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile without secret: %v", err)
	}
	ready := readyCondition(ctx, t, c, key)
	if ready.Status != metav1.ConditionFalse {
		t.Fatalf("missing secret must not be Ready, got %s", ready.Status)
	}
	if ready.Reason != "ValidationFailed" {
		t.Fatalf("unexpected reason %q", ready.Reason)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook-credentials", Namespace: namespace},
		Data:       map[string][]byte{"token": []byte("s3cr3t")},
	}
	if err := c.Create(ctx, secret); err != nil {
		t.Fatalf("create secret: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile with secret: %v", err)
	}
	ready = readyCondition(ctx, t, c, key)
	if ready.Status != metav1.ConditionTrue {
		t.Fatalf("resolvable credential must be Ready, got %s: %s", ready.Status, ready.Message)
	}

	var reconciled kickv1alpha1.NotificationPolicy
	if err := c.Get(ctx, key, &reconciled); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if reconciled.Status.ObservedGeneration != reconciled.Generation {
		t.Fatalf("observedGeneration %d does not track generation %d",
			reconciled.Status.ObservedGeneration, reconciled.Generation)
	}
	if reconciled.Status.Delivered != 0 || reconciled.Status.Failed != 0 {
		t.Fatal("validation must not deliver anything")
	}
}

func readyCondition(ctx context.Context, t *testing.T, c client.Client, key types.NamespacedName) metav1.Condition {
	t.Helper()
	var policy kickv1alpha1.NotificationPolicy
	if err := c.Get(ctx, key, &policy); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	condition := apimeta.FindStatusCondition(policy.Status.Conditions, controller.NotificationPolicyReadyCondition)
	if condition == nil {
		t.Fatal("policy has no Ready condition")
	}
	return *condition
}
