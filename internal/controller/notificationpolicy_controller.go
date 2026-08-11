package controller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
)

const (
	// NotificationPolicyReadyCondition reports whether a policy could deliver.
	NotificationPolicyReadyCondition = "Ready"

	notificationPolicyReasonValidated = "Validated"
	notificationPolicyReasonInvalid   = "ValidationFailed"
)

// NotificationPolicyReconciler validates a NotificationPolicy against the
// Secrets it references and reports the result. It never delivers anything, and
// a policy that fails validation is still attempted at delivery time: a broken
// notification must not silently disable itself.
type NotificationPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NotificationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var policy kickv1alpha1.NotificationPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	condition := metav1.Condition{
		Type:               NotificationPolicyReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             notificationPolicyReasonValidated,
		Message:            "webhook credentials resolve",
		ObservedGeneration: policy.Generation,
	}
	if err := r.validate(ctx, &policy); err != nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = notificationPolicyReasonInvalid
		condition.Message = err.Error()
	}

	patched := policy.DeepCopy()
	patched.Status.ObservedGeneration = policy.Generation
	changed := apimeta.SetStatusCondition(&patched.Status.Conditions, condition)
	if !changed && patched.Status.ObservedGeneration == policy.Status.ObservedGeneration {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.Status().Update(ctx, patched)
}

// validate resolves every Secret the webhook needs. The error message names
// Secrets and keys but never their values.
func (r *NotificationPolicyReconciler) validate(ctx context.Context, policy *kickv1alpha1.NotificationPolicy) error {
	webhook := policy.Spec.Webhook
	for _, header := range webhook.Headers {
		if header.ValueFrom == nil {
			continue
		}
		if _, err := r.secretValue(ctx, policy.Namespace, *header.ValueFrom); err != nil {
			return fmt.Errorf("header %s: %w", header.Name, err)
		}
	}
	if err := r.validateAuth(ctx, policy.Namespace, webhook.Auth); err != nil {
		return err
	}
	return r.validateTLS(ctx, policy.Namespace, webhook.TLS)
}

func (r *NotificationPolicyReconciler) validateAuth(ctx context.Context, namespace string, auth *kickv1alpha1.NotificationAuth) error {
	if auth == nil {
		return nil
	}
	if auth.BearerToken != nil && auth.Basic != nil {
		return errors.New("webhook auth sets both bearerToken and basic")
	}
	switch {
	case auth.BearerToken != nil:
		if _, err := r.secretValue(ctx, namespace, *auth.BearerToken); err != nil {
			return fmt.Errorf("auth.bearerToken: %w", err)
		}
	case auth.Basic != nil:
		if _, err := r.secretValue(ctx, namespace, auth.Basic.Username); err != nil {
			return fmt.Errorf("auth.basic.username: %w", err)
		}
		if _, err := r.secretValue(ctx, namespace, auth.Basic.Password); err != nil {
			return fmt.Errorf("auth.basic.password: %w", err)
		}
	}
	return nil
}

func (r *NotificationPolicyReconciler) validateTLS(ctx context.Context, namespace string, tlsSpec *kickv1alpha1.NotificationTLS) error {
	if tlsSpec == nil {
		return nil
	}
	if tlsSpec.CABundle != nil {
		pem, err := r.secretValue(ctx, namespace, *tlsSpec.CABundle)
		if err != nil {
			return fmt.Errorf("tls.caBundle: %w", err)
		}
		if !x509.NewCertPool().AppendCertsFromPEM([]byte(pem)) {
			return errors.New("tls.caBundle contains no valid certificate")
		}
	}
	if tlsSpec.ClientCertificate == nil {
		return nil
	}
	var secret corev1.Secret
	key := types.NamespacedName{Namespace: namespace, Name: tlsSpec.ClientCertificate.Name}
	if err := r.Get(ctx, key, &secret); err != nil {
		return fmt.Errorf("tls.clientCertificate: %w", err)
	}
	if _, err := tls.X509KeyPair(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey]); err != nil {
		return errors.New("tls.clientCertificate secret does not contain a usable tls.crt/tls.key pair")
	}
	return nil
}

func (r *NotificationPolicyReconciler) secretValue(ctx context.Context, namespace string, ref kickv1alpha1.SecretKeyRef) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", err
	}
	value, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("secret %s has no key %s", ref.Name, ref.Key)
	}
	return string(value), nil
}

func (r *NotificationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("notificationpolicy").
		For(&kickv1alpha1.NotificationPolicy{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.mapSecretToPolicies)).
		Complete(r)
}

// mapSecretToPolicies re-validates the policies that reference a Secret, so a
// policy becomes Ready as soon as its missing credential is created.
func (r *NotificationPolicyReconciler) mapSecretToPolicies(ctx context.Context, obj client.Object) []reconcile.Request {
	var policies kickv1alpha1.NotificationPolicyList
	if err := r.List(ctx, &policies, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for i := range policies.Items {
		policy := &policies.Items[i]
		if !policyReferencesSecret(policy, obj.GetName()) {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name},
		})
	}
	return requests
}

func policyReferencesSecret(policy *kickv1alpha1.NotificationPolicy, name string) bool {
	for _, ref := range referencedSecrets(policy) {
		if ref == name {
			return true
		}
	}
	return false
}

func referencedSecrets(policy *kickv1alpha1.NotificationPolicy) []string {
	webhook := policy.Spec.Webhook
	var names []string
	for _, header := range webhook.Headers {
		if header.ValueFrom != nil {
			names = append(names, header.ValueFrom.Name)
		}
	}
	if auth := webhook.Auth; auth != nil {
		if auth.BearerToken != nil {
			names = append(names, auth.BearerToken.Name)
		}
		if auth.Basic != nil {
			names = append(names, auth.Basic.Username.Name, auth.Basic.Password.Name)
		}
	}
	if tlsSpec := webhook.TLS; tlsSpec != nil {
		if tlsSpec.CABundle != nil {
			names = append(names, tlsSpec.CABundle.Name)
		}
		if tlsSpec.ClientCertificate != nil {
			names = append(names, tlsSpec.ClientCertificate.Name)
		}
	}
	return names
}
