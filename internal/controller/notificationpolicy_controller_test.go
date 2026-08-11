package controller

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
)

const notificationTestNamespace = "team-a"

func notificationPolicy(name string, webhook kickv1alpha1.NotificationWebhook) *kickv1alpha1.NotificationPolicy {
	return &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: name},
		Spec:       kickv1alpha1.NotificationPolicySpec{Webhook: webhook},
	}
}

func reconcileNotificationPolicy(t *testing.T, c client.Client, scheme *runtime.Scheme, name string) kickv1alpha1.NotificationPolicy {
	t.Helper()
	r := &NotificationPolicyReconciler{Client: c, Scheme: scheme}
	key := types.NamespacedName{Namespace: notificationTestNamespace, Name: name}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var got kickv1alpha1.NotificationPolicy
	if err := c.Get(context.Background(), key, &got); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	return got
}

func TestNotificationPolicyReadyWhenCredentialsResolve(t *testing.T) {
	scheme := testScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-creds"},
		Data: map[string][]byte{
			"token":    []byte("header-value"),
			"username": []byte("kick"),
			"password": []byte("s3cr3t-value"),
		},
	}
	policy := notificationPolicy("chatops", kickv1alpha1.NotificationWebhook{
		URL:    "https://hooks.example.com/kick",
		Method: "POST",
		Headers: []kickv1alpha1.NotificationHeader{
			{Name: "X-Route", Value: "team-a"},
			{Name: "X-Token", ValueFrom: &kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "token"}},
		},
		Auth: &kickv1alpha1.NotificationAuth{Basic: &kickv1alpha1.NotificationBasicAuth{
			Username: kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "username"},
			Password: kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "password"},
		}},
	})

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&kickv1alpha1.NotificationPolicy{}).
		WithObjects(secret, policy).Build()

	got := reconcileNotificationPolicy(t, c, scheme, "chatops")
	cond := apimeta.FindStatusCondition(got.Status.Conditions, NotificationPolicyReadyCondition)
	if cond == nil {
		t.Fatalf("no %s condition", NotificationPolicyReadyCondition)
	}
	if cond.Status != metav1.ConditionTrue {
		t.Fatalf("Ready = %s (%s: %s), want True", cond.Status, cond.Reason, cond.Message)
	}
	if cond.Reason != "Validated" {
		t.Fatalf("reason = %s, want Validated", cond.Reason)
	}
	if got.Status.Delivered != 0 || got.Status.Failed != 0 {
		t.Fatalf("validation wrote delivery counters: delivered=%d failed=%d", got.Status.Delivered, got.Status.Failed)
	}
	if got.Status.LastDeliveryTime != nil {
		t.Fatalf("validation set lastDeliveryTime: %v", got.Status.LastDeliveryTime)
	}
}

func TestNotificationPolicyNotReadyWhenSecretUnresolvable(t *testing.T) {
	tests := []struct {
		name          string
		withSecret    bool
		wantSubstring []string
	}{
		{
			name:          "secret exists without the referenced key",
			withSecret:    true,
			wantSubstring: []string{"hook-creds", "token"},
		},
		{
			name:          "secret does not exist",
			withSecret:    false,
			wantSubstring: []string{"hook-creds"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testScheme(t)
			objects := []client.Object{notificationPolicy("chatops", kickv1alpha1.NotificationWebhook{
				URL: "https://hooks.example.com/kick",
				Headers: []kickv1alpha1.NotificationHeader{
					{Name: "X-Token", ValueFrom: &kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "token"}},
				},
			})}
			if tt.withSecret {
				objects = append(objects, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-creds"},
					Data:       map[string][]byte{"other": []byte("s3cr3t-value")},
				})
			}

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&kickv1alpha1.NotificationPolicy{}).
				WithObjects(objects...).Build()

			got := reconcileNotificationPolicy(t, c, scheme, "chatops")
			cond := apimeta.FindStatusCondition(got.Status.Conditions, NotificationPolicyReadyCondition)
			if cond == nil {
				t.Fatalf("no %s condition", NotificationPolicyReadyCondition)
			}
			if cond.Status != metav1.ConditionFalse {
				t.Fatalf("Ready = %s, want False", cond.Status)
			}
			if cond.Reason != "ValidationFailed" {
				t.Fatalf("reason = %s, want ValidationFailed", cond.Reason)
			}
			for _, want := range tt.wantSubstring {
				if !strings.Contains(cond.Message, want) {
					t.Fatalf("message %q does not mention %q", cond.Message, want)
				}
			}
			if strings.Contains(cond.Message, "s3cr3t-value") {
				t.Fatalf("message leaks a secret value: %q", cond.Message)
			}
			if got.Status.Delivered != 0 || got.Status.Failed != 0 {
				t.Fatalf("validation wrote delivery counters: delivered=%d failed=%d", got.Status.Delivered, got.Status.Failed)
			}
		})
	}
}

func TestNotificationPolicyAuthAndTLSValidation(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kick-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	credentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-creds"},
		Data: map[string][]byte{
			"token":    []byte("t"),
			"username": []byte("kick"),
			"password": []byte("s3cr3t-value"),
		},
	}

	tests := []struct {
		name        string
		secrets     []client.Object
		webhook     kickv1alpha1.NotificationWebhook
		wantReady   bool
		wantMessage string
	}{
		{
			name:    "both auth schemes set",
			secrets: []client.Object{credentials},
			webhook: kickv1alpha1.NotificationWebhook{
				URL: "https://hooks.example.com/kick",
				Auth: &kickv1alpha1.NotificationAuth{
					BearerToken: &kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "token"},
					Basic: &kickv1alpha1.NotificationBasicAuth{
						Username: kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "username"},
						Password: kickv1alpha1.SecretKeyRef{Name: "hook-creds", Key: "password"},
					},
				},
			},
			wantMessage: "both bearerToken and basic",
		},
		{
			name: "ca bundle is not a certificate",
			secrets: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-ca"},
				Data:       map[string][]byte{"ca.crt": []byte("definitely not a certificate")},
			}},
			webhook: kickv1alpha1.NotificationWebhook{
				URL: "https://hooks.example.com/kick",
				TLS: &kickv1alpha1.NotificationTLS{
					CABundle: &kickv1alpha1.SecretKeyRef{Name: "hook-ca", Key: "ca.crt"},
				},
			},
			wantMessage: "tls.caBundle contains no valid certificate",
		},
		{
			name: "ca bundle is a valid pem certificate",
			secrets: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-ca"},
				Data:       map[string][]byte{"ca.crt": caPEM},
			}},
			webhook: kickv1alpha1.NotificationWebhook{
				URL: "https://hooks.example.com/kick",
				TLS: &kickv1alpha1.NotificationTLS{
					CABundle: &kickv1alpha1.SecretKeyRef{Name: "hook-ca", Key: "ca.crt"},
				},
			},
			wantReady: true,
		},
		{
			name: "client certificate secret has no usable key pair",
			secrets: []client.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: notificationTestNamespace, Name: "hook-mtls"},
				Type:       corev1.SecretTypeTLS,
				Data:       map[string][]byte{corev1.TLSCertKey: caPEM, corev1.TLSPrivateKeyKey: []byte("s3cr3t-value")},
			}},
			webhook: kickv1alpha1.NotificationWebhook{
				URL: "https://hooks.example.com/kick",
				TLS: &kickv1alpha1.NotificationTLS{
					ClientCertificate: &kickv1alpha1.SecretKeyRef{Name: "hook-mtls", Key: "tls.crt"},
				},
			},
			wantMessage: "usable tls.crt/tls.key pair",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := testScheme(t)
			objects := append([]client.Object{notificationPolicy("chatops", tt.webhook)}, tt.secrets...)
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&kickv1alpha1.NotificationPolicy{}).
				WithObjects(objects...).Build()

			got := reconcileNotificationPolicy(t, c, scheme, "chatops")
			cond := apimeta.FindStatusCondition(got.Status.Conditions, NotificationPolicyReadyCondition)
			if cond == nil {
				t.Fatalf("no %s condition", NotificationPolicyReadyCondition)
			}
			if tt.wantReady {
				if cond.Status != metav1.ConditionTrue {
					t.Fatalf("Ready = %s (%s: %s), want True", cond.Status, cond.Reason, cond.Message)
				}
				if cond.Reason != "Validated" {
					t.Fatalf("reason = %s, want Validated", cond.Reason)
				}
				return
			}
			if cond.Status != metav1.ConditionFalse {
				t.Fatalf("Ready = %s, want False", cond.Status)
			}
			if cond.Reason != "ValidationFailed" {
				t.Fatalf("reason = %s, want ValidationFailed", cond.Reason)
			}
			if !strings.Contains(cond.Message, tt.wantMessage) {
				t.Fatalf("message %q does not contain %q", cond.Message, tt.wantMessage)
			}
			if strings.Contains(cond.Message, "s3cr3t-value") {
				t.Fatalf("message leaks a secret value: %q", cond.Message)
			}
		})
	}
}

func TestNotificationPolicyReconcileMissingPolicy(t *testing.T) {
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&kickv1alpha1.NotificationPolicy{}).Build()

	r := &NotificationPolicyReconciler{Client: c, Scheme: scheme}
	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: notificationTestNamespace, Name: "gone"},
	})
	if err != nil {
		t.Fatalf("reconcile missing policy: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %s", result.RequeueAfter)
	}
}

func TestNotificationPolicyReferencedSecrets(t *testing.T) {
	policy := notificationPolicy("chatops", kickv1alpha1.NotificationWebhook{
		URL: "https://hooks.example.com/kick",
		Headers: []kickv1alpha1.NotificationHeader{
			{Name: "X-Route", Value: "team-a"},
			{Name: "X-Token", ValueFrom: &kickv1alpha1.SecretKeyRef{Name: "header-secret", Key: "token"}},
		},
		Auth: &kickv1alpha1.NotificationAuth{
			BearerToken: &kickv1alpha1.SecretKeyRef{Name: "bearer-secret", Key: "token"},
			Basic: &kickv1alpha1.NotificationBasicAuth{
				Username: kickv1alpha1.SecretKeyRef{Name: "username-secret", Key: "username"},
				Password: kickv1alpha1.SecretKeyRef{Name: "password-secret", Key: "password"},
			},
		},
		TLS: &kickv1alpha1.NotificationTLS{
			CABundle:          &kickv1alpha1.SecretKeyRef{Name: "ca-secret", Key: "ca.crt"},
			ClientCertificate: &kickv1alpha1.SecretKeyRef{Name: "client-cert-secret", Key: "tls.crt"},
		},
	})

	want := []string{"bearer-secret", "ca-secret", "client-cert-secret", "header-secret", "password-secret", "username-secret"}
	got := referencedSecrets(policy)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("referencedSecrets = %v, want %v", got, want)
	}

	for _, name := range want {
		if !policyReferencesSecret(policy, name) {
			t.Fatalf("policyReferencesSecret(%q) = false, want true", name)
		}
	}
	if policyReferencesSecret(policy, "unrelated-secret") {
		t.Fatalf("policyReferencesSecret(unrelated-secret) = true, want false")
	}

	bare := notificationPolicy("bare", kickv1alpha1.NotificationWebhook{
		URL:     "https://hooks.example.com/kick",
		Headers: []kickv1alpha1.NotificationHeader{{Name: "X-Route", Value: "team-a"}},
	})
	if refs := referencedSecrets(bare); len(refs) != 0 {
		t.Fatalf("referencedSecrets for a policy without secret refs = %v, want none", refs)
	}
}
