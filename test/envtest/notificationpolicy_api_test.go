package envtest

import (
	"context"
	"path/filepath"
	"testing"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestNotificationPolicyValidationEnvtest(t *testing.T) {
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
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "notifications"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	valid := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "notifications"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Suspend:          true,
			Phases:           []kickv1alpha1.KickRequestPhase{kickv1alpha1.KickRequestPhaseSucceeded, kickv1alpha1.KickRequestPhaseDryRun},
			WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payments-api"}},
			Webhook: kickv1alpha1.NotificationWebhook{
				URL:            "https://alerts.example.com/hooks/kick",
				Method:         "PUT",
				TimeoutSeconds: 42,
				Headers: []kickv1alpha1.NotificationHeader{
					{Name: "X-Kick-Source", Value: "production"},
					{Name: "X-Kick-Tenant", ValueFrom: &kickv1alpha1.SecretKeyRef{Name: "kick-webhook", Key: "tenant"}},
				},
				Auth: &kickv1alpha1.NotificationAuth{
					BearerToken: &kickv1alpha1.SecretKeyRef{Name: "kick-webhook", Key: "token"},
					Basic: &kickv1alpha1.NotificationBasicAuth{
						Username: kickv1alpha1.SecretKeyRef{Name: "kick-webhook", Key: "username"},
						Password: kickv1alpha1.SecretKeyRef{Name: "kick-webhook", Key: "password"},
					},
				},
				TLS: &kickv1alpha1.NotificationTLS{
					CABundle:          &kickv1alpha1.SecretKeyRef{Name: "kick-webhook-tls", Key: "ca.crt"},
					ClientCertificate: &kickv1alpha1.SecretKeyRef{Name: "kick-webhook-tls", Key: "tls.crt"},
				},
			},
		},
	}
	if err := c.Create(ctx, valid); err != nil {
		t.Fatalf("create valid notificationpolicy: %v", err)
	}

	var persisted kickv1alpha1.NotificationPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(valid), &persisted); err != nil {
		t.Fatalf("get notificationpolicy: %v", err)
	}
	if !persisted.Spec.Suspend {
		t.Fatal("suspend was not persisted")
	}
	if len(persisted.Spec.Phases) != 2 || persisted.Spec.Phases[1] != kickv1alpha1.KickRequestPhaseDryRun {
		t.Fatalf("phases were not persisted: %v", persisted.Spec.Phases)
	}
	if persisted.Spec.WorkloadSelector == nil {
		t.Fatal("workloadSelector was not persisted")
	}
	if persisted.Spec.Webhook.Method != "PUT" || persisted.Spec.Webhook.TimeoutSeconds != 42 {
		t.Fatalf("webhook settings were not persisted: %+v", persisted.Spec.Webhook)
	}
	if len(persisted.Spec.Webhook.Headers) != 2 || persisted.Spec.Webhook.Headers[1].ValueFrom == nil {
		t.Fatalf("headers were not persisted: %+v", persisted.Spec.Webhook.Headers)
	}
	if persisted.Spec.Webhook.Auth == nil || persisted.Spec.Webhook.Auth.BearerToken == nil || persisted.Spec.Webhook.Auth.Basic == nil {
		t.Fatalf("auth was not persisted: %+v", persisted.Spec.Webhook.Auth)
	}
	if persisted.Spec.Webhook.TLS == nil || persisted.Spec.Webhook.TLS.CABundle == nil || persisted.Spec.Webhook.TLS.ClientCertificate == nil {
		t.Fatalf("tls was not persisted: %+v", persisted.Spec.Webhook.TLS)
	}

	// Defaults must be applied by the API server, not only by the dispatcher.
	defaulted := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "defaults", Namespace: "notifications"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: "http://alerts.example.com/hooks/kick"},
		},
	}
	if err := c.Create(ctx, defaulted); err != nil {
		t.Fatalf("create defaulted notificationpolicy: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(defaulted), &persisted); err != nil {
		t.Fatalf("get defaulted notificationpolicy: %v", err)
	}
	if persisted.Spec.Webhook.Method != "POST" {
		t.Fatalf("method default = %q, want POST", persisted.Spec.Webhook.Method)
	}
	if persisted.Spec.Webhook.TimeoutSeconds != 10 {
		t.Fatalf("timeoutSeconds default = %d, want 10", persisted.Spec.Webhook.TimeoutSeconds)
	}

	// A non-HTTP URL must be rejected so credentials are never sent to an
	// arbitrary scheme.
	invalid := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-url", Namespace: "notifications"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: "ftp://alerts.example.com/hooks/kick"},
		},
	}
	if err := c.Create(ctx, invalid); err == nil {
		t.Fatal("expected a non-http url to be rejected")
	}

	invalidMethod := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-method", Namespace: "notifications"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: "https://alerts.example.com/hooks/kick", Method: "DELETE"},
		},
	}
	if err := c.Create(ctx, invalidMethod); err == nil {
		t.Fatal("expected an unsupported method to be rejected")
	}

	invalidTimeout := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-timeout", Namespace: "notifications"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: "https://alerts.example.com/hooks/kick", TimeoutSeconds: 500},
		},
	}
	if err := c.Create(ctx, invalidTimeout); err == nil {
		t.Fatal("expected an out-of-range timeout to be rejected")
	}
}

func TestNotificationPolicyStatusEnvtest(t *testing.T) {
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
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "notifications-status"}}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}

	policy := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "notifications-status"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: "https://alerts.example.com/hooks/kick"},
		},
	}
	if err := c.Create(ctx, policy); err != nil {
		t.Fatalf("create notificationpolicy: %v", err)
	}

	now := metav1.Now()
	policy.Status = kickv1alpha1.NotificationPolicyStatus{
		ObservedGeneration: policy.Generation,
		LastDeliveryTime:   &now,
		LastError:          "502 from endpoint",
		Delivered:          7,
		Failed:             2,
		Conditions: []metav1.Condition{{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "DeliveryFailed",
			Message:            "endpoint returned 502",
			LastTransitionTime: now,
			ObservedGeneration: policy.Generation,
		}},
	}
	if err := c.Status().Update(ctx, policy); err != nil {
		t.Fatalf("update status: %v", err)
	}

	var persisted kickv1alpha1.NotificationPolicy
	if err := c.Get(ctx, client.ObjectKeyFromObject(policy), &persisted); err != nil {
		t.Fatalf("get notificationpolicy: %v", err)
	}
	if persisted.Status.ObservedGeneration != policy.Generation {
		t.Fatalf("observedGeneration = %d", persisted.Status.ObservedGeneration)
	}
	if persisted.Status.LastDeliveryTime == nil {
		t.Fatal("lastDeliveryTime was not persisted")
	}
	if persisted.Status.LastError != "502 from endpoint" {
		t.Fatalf("lastError = %q", persisted.Status.LastError)
	}
	if persisted.Status.Delivered != 7 || persisted.Status.Failed != 2 {
		t.Fatalf("counters = %d/%d, want 7/2", persisted.Status.Delivered, persisted.Status.Failed)
	}
	if len(persisted.Status.Conditions) != 1 {
		t.Fatalf("conditions = %v", persisted.Status.Conditions)
	}
}
