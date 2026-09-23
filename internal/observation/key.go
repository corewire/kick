package observation

import (
	"context"
	"crypto/rand"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// FingerprintKeySecretName is the Secret in the manager namespace that holds
	// the installation-private fingerprint key.
	FingerprintKeySecretName = "kick-fingerprint-key"
	fingerprintKeyField      = "key"
	fingerprintKeyLength     = 32
)

// LoadOrCreateFingerprintKey returns the installation-private fingerprint key,
// generating and persisting it on first start. Concurrent replicas racing on
// creation converge on whichever Secret won; deleting the Secret rotates the
// key, which re-anchors every observation record on its next observation (see
// Observer.observe) rather than restarting anything.
func LoadOrCreateFingerprintKey(ctx context.Context, c client.Client, namespace string) ([]byte, error) {
	key := types.NamespacedName{Namespace: namespace, Name: FingerprintKeySecretName}
	var secret corev1.Secret
	err := c.Get(ctx, key, &secret)
	if err == nil {
		return fingerprintKeyFrom(&secret)
	}
	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("read fingerprint key %s: %w", key, err)
	}

	fresh := make([]byte, fingerprintKeyLength)
	if _, err := rand.Read(fresh); err != nil {
		return nil, fmt.Errorf("generate fingerprint key: %w", err)
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{fingerprintKeyField: fresh},
	}
	if err := c.Create(ctx, &secret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("create fingerprint key %s: %w", key, err)
		}
		if err := c.Get(ctx, key, &secret); err != nil {
			return nil, fmt.Errorf("read fingerprint key %s after create race: %w", key, err)
		}
	}
	return fingerprintKeyFrom(&secret)
}

func fingerprintKeyFrom(secret *corev1.Secret) ([]byte, error) {
	value := secret.Data[fingerprintKeyField]
	if len(value) < fingerprintKeyLength {
		return nil, fmt.Errorf("fingerprint key %s/%s: field %q missing or shorter than %d bytes", secret.Namespace, secret.Name, fingerprintKeyField, fingerprintKeyLength)
	}
	return value, nil
}
