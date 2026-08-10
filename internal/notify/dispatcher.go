// Package notify delivers KickRequest phase transitions to webhook endpoints
// described by NotificationPolicy objects.
//
// Delivery is strictly best-effort and fully decoupled from reconciliation: a
// slow or broken endpoint must never delay or fail a restart. Events are queued
// on a bounded channel and dropped (oldest first) when the queue is full.
package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DefaultQueueSize bounds in-flight events. A burst larger than this drops the
// oldest events rather than growing memory without limit.
const DefaultQueueSize = 256

// maxAttempts bounds retries per event so a permanently broken endpoint cannot
// occupy a worker indefinitely.
const maxAttempts = 3

// baseBackoff is the first retry delay; it doubles per attempt.
const baseBackoff = 500 * time.Millisecond

// defaultTerminalPhases are notified when a policy does not list phases.
var defaultTerminalPhases = []kickv1alpha1.KickRequestPhase{
	kickv1alpha1.KickRequestPhaseSucceeded,
	kickv1alpha1.KickRequestPhaseFailed,
	kickv1alpha1.KickRequestPhaseNoLongerRequired,
	kickv1alpha1.KickRequestPhaseDryRun,
}

// Event is the provider-neutral description of a KickRequest transition.
//
// It deliberately carries no dependency content, no Secret data and no content
// digests: only object identity, phase and reason.
type Event struct {
	Namespace      string            `json:"namespace"`
	RequestName    string            `json:"requestName"`
	Phase          string            `json:"phase"`
	Reason         string            `json:"reason"`
	Message        string            `json:"message"`
	TargetKind     string            `json:"targetKind"`
	TargetName     string            `json:"targetName"`
	GitOpsProvider string            `json:"gitOpsProvider,omitempty"`
	OccurredAt     time.Time         `json:"occurredAt"`
	workloadLabels map[string]string `json:"-"`
}

// WithWorkloadLabels attaches the target workload labels used for policy
// selection. They are never serialized into the payload.
func (e Event) WithWorkloadLabels(l map[string]string) Event {
	e.workloadLabels = l
	return e
}

// Dispatcher is the notification sink used by the controllers.
type Dispatcher interface {
	Notify(Event)
}

// Noop discards every event. It is the default when notifications are disabled.
type Noop struct{}

func (Noop) Notify(Event) {}

// WebhookDispatcher resolves NotificationPolicy objects at delivery time and
// posts matching events.
type WebhookDispatcher struct {
	client    client.Client
	queue     chan Event
	transport func(*NotificationTransport) *http.Client
	dropped   func()
}

// NotificationTransport carries the resolved TLS material for one endpoint.
type NotificationTransport struct {
	RootCAs     *x509.CertPool
	Certificate *tls.Certificate
	Timeout     time.Duration
}

// NewWebhookDispatcher returns a dispatcher with a bounded queue. Start must be
// called to drain it.
func NewWebhookDispatcher(c client.Client, queueSize int) *WebhookDispatcher {
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}
	return &WebhookDispatcher{
		client:    c,
		queue:     make(chan Event, queueSize),
		transport: newHTTPClient,
	}
}

// Notify enqueues an event. It never blocks: when the queue is full the oldest
// event is discarded so that reconciliation is never slowed down by delivery.
func (d *WebhookDispatcher) Notify(event Event) {
	if d == nil {
		return
	}
	for {
		select {
		case d.queue <- event:
			return
		default:
		}
		select {
		case <-d.queue:
			observeDropped()
			if d.dropped != nil {
				d.dropped()
			}
		default:
			// Another worker drained the queue between the two selects; retry.
		}
	}
}

// Start drains the queue until the context is cancelled. It satisfies
// manager.Runnable.
func (d *WebhookDispatcher) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-d.queue:
			d.deliver(ctx, event)
		}
	}
}

// NeedLeaderElection keeps a single replica from duplicating notifications.
func (d *WebhookDispatcher) NeedLeaderElection() bool { return true }

func (d *WebhookDispatcher) deliver(ctx context.Context, event Event) {
	logger := log.FromContext(ctx).WithName("notify")

	var policies kickv1alpha1.NotificationPolicyList
	if err := d.client.List(ctx, &policies, client.InNamespace(event.Namespace)); err != nil {
		logger.Error(err, "listing notification policies failed", "namespace", event.Namespace)
		return
	}

	body, err := json.Marshal(event)
	if err != nil {
		logger.Error(err, "encoding notification payload failed")
		return
	}

	for i := range policies.Items {
		policy := &policies.Items[i]
		matches, err := policyMatches(policy, event)
		if err != nil {
			logger.Error(err, "evaluating notification policy failed", "policy", policy.Name)
			continue
		}
		if !matches {
			continue
		}
		if err := d.send(ctx, policy, body); err != nil {
			// Only the error is recorded; request and response bodies may echo
			// sensitive routing information.
			logger.Error(err, "notification delivery failed", "policy", policy.Name)
			observeDelivery(policy.Namespace, policy.Name, false)
			d.recordStatus(ctx, policy, err)
			continue
		}
		observeDelivery(policy.Namespace, policy.Name, true)
		d.recordStatus(ctx, policy, nil)
	}
}

func (d *WebhookDispatcher) send(ctx context.Context, policy *kickv1alpha1.NotificationPolicy, body []byte) error {
	transport, err := d.resolveTransport(ctx, policy)
	if err != nil {
		return err
	}
	header, err := d.resolveHeaders(ctx, policy)
	if err != nil {
		return err
	}
	httpClient := d.transport(transport)

	method := policy.Spec.Webhook.Method
	if method == "" {
		method = http.MethodPost
	}

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(baseBackoff << (attempt - 1)):
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, policy.Spec.Webhook.URL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header = header.Clone()
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("endpoint returned status %d", resp.StatusCode)
		// 4xx other than 429 will not succeed on retry.
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return lastErr
		}
	}
	return lastErr
}

func (d *WebhookDispatcher) resolveHeaders(ctx context.Context, policy *kickv1alpha1.NotificationPolicy) (http.Header, error) {
	header := http.Header{}
	for _, h := range policy.Spec.Webhook.Headers {
		switch {
		case h.ValueFrom != nil:
			value, err := d.secretValue(ctx, policy.Namespace, *h.ValueFrom)
			if err != nil {
				return nil, err
			}
			header.Set(h.Name, value)
		case h.Value != "":
			header.Set(h.Name, h.Value)
		}
	}

	auth := policy.Spec.Webhook.Auth
	if auth == nil {
		return header, nil
	}
	if auth.BearerToken != nil && auth.Basic != nil {
		return nil, errors.New("webhook auth sets both bearerToken and basic")
	}
	switch {
	case auth.BearerToken != nil:
		token, err := d.secretValue(ctx, policy.Namespace, *auth.BearerToken)
		if err != nil {
			return nil, err
		}
		header.Set("Authorization", "Bearer "+token)
	case auth.Basic != nil:
		user, err := d.secretValue(ctx, policy.Namespace, auth.Basic.Username)
		if err != nil {
			return nil, err
		}
		password, err := d.secretValue(ctx, policy.Namespace, auth.Basic.Password)
		if err != nil {
			return nil, err
		}
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+password)))
	}
	return header, nil
}

func (d *WebhookDispatcher) resolveTransport(ctx context.Context, policy *kickv1alpha1.NotificationPolicy) (*NotificationTransport, error) {
	timeout := time.Duration(policy.Spec.Webhook.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := &NotificationTransport{Timeout: timeout}

	tlsSpec := policy.Spec.Webhook.TLS
	if tlsSpec == nil {
		return transport, nil
	}
	if tlsSpec.CABundle != nil {
		pem, err := d.secretValue(ctx, policy.Namespace, *tlsSpec.CABundle)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			return nil, errors.New("caBundle contains no valid certificate")
		}
		transport.RootCAs = pool
	}
	if tlsSpec.ClientCertificate != nil {
		var secret corev1.Secret
		key := types.NamespacedName{Namespace: policy.Namespace, Name: tlsSpec.ClientCertificate.Name}
		if err := d.client.Get(ctx, key, &secret); err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
		if err != nil {
			return nil, errors.New("clientCertificate secret does not contain a usable tls.crt/tls.key pair")
		}
		transport.Certificate = &cert
	}
	return transport, nil
}

func (d *WebhookDispatcher) secretValue(ctx context.Context, namespace string, ref kickv1alpha1.SecretKeyRef) (string, error) {
	var secret corev1.Secret
	if err := d.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", err
	}
	value, ok := secret.Data[ref.Key]
	if !ok {
		// The key name is safe to log; the value is not.
		return "", fmt.Errorf("secret %s has no key %s", ref.Name, ref.Key)
	}
	return string(value), nil
}

// recordStatus records the outcome on the policy. Status update failures are
// intentionally ignored: they must not retry or block delivery.
func (d *WebhookDispatcher) recordStatus(ctx context.Context, policy *kickv1alpha1.NotificationPolicy, deliveryErr error) {
	now := metav1.Now()
	patched := policy.DeepCopy()
	patched.Status.ObservedGeneration = policy.Generation
	if deliveryErr == nil {
		patched.Status.Delivered++
		patched.Status.LastDeliveryTime = &now
		patched.Status.LastError = ""
	} else {
		patched.Status.Failed++
		patched.Status.LastError = deliveryErr.Error()
	}
	_ = d.client.Status().Update(ctx, patched)
}

// policyMatches reports whether a policy wants this event.
func policyMatches(policy *kickv1alpha1.NotificationPolicy, event Event) (bool, error) {
	if policy.Spec.Suspend {
		return false, nil
	}
	if !phaseSelected(policy.Spec.Phases, event.Phase) {
		return false, nil
	}
	if policy.Spec.WorkloadSelector == nil {
		return true, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(policy.Spec.WorkloadSelector)
	if err != nil {
		return false, err
	}
	return selector.Matches(labels.Set(event.workloadLabels)), nil
}

func phaseSelected(configured []kickv1alpha1.KickRequestPhase, phase string) bool {
	if len(configured) == 0 {
		configured = defaultTerminalPhases
	}
	for _, candidate := range configured {
		if string(candidate) == phase {
			return true
		}
	}
	return false
}

func newHTTPClient(transport *NotificationTransport) *http.Client {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if transport.RootCAs != nil {
		tlsConfig.RootCAs = transport.RootCAs
	}
	if transport.Certificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*transport.Certificate}
	}
	return &http.Client{
		Timeout:   transport.Timeout,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}
}
