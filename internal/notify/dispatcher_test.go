package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding core scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding kick scheme: %v", err)
	}
	return scheme
}

func testEvent() Event {
	return Event{
		Namespace:   "prod",
		RequestName: "web-1",
		Phase:       string(kickv1alpha1.KickRequestPhaseSucceeded),
		Reason:      "RestartCompleted",
		Message:     "restart completed",
		TargetKind:  "Deployment",
		TargetName:  "web",
		OccurredAt:  time.Unix(0, 0).UTC(),
	}
}

// The payload must never leak dependency content, Secret data or digests.
func TestEventPayloadCarriesNoSecretMaterial(t *testing.T) {
	body, err := json.Marshal(testEvent().WithWorkloadLabels(map[string]string{"app": "web"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowed := map[string]struct{}{
		"namespace": {}, "requestName": {}, "phase": {}, "reason": {}, "message": {},
		"targetKind": {}, "targetName": {}, "gitOpsProvider": {}, "occurredAt": {},
	}
	for key := range payload {
		if _, ok := allowed[key]; !ok {
			t.Fatalf("unexpected field %q in notification payload", key)
		}
	}
	for _, forbidden := range []string{"digest", "hash", "data", "value", "labels"} {
		if strings.Contains(strings.ToLower(string(body)), forbidden) {
			t.Fatalf("notification payload must not contain %q: %s", forbidden, body)
		}
	}
}

func TestPhaseSelectedDefaultsToTerminalPhases(t *testing.T) {
	if !phaseSelected(nil, string(kickv1alpha1.KickRequestPhaseSucceeded)) {
		t.Fatal("Succeeded must be notified by default")
	}
	if !phaseSelected(nil, string(kickv1alpha1.KickRequestPhaseDryRun)) {
		t.Fatal("DryRun must be notified by default")
	}
	if phaseSelected(nil, string(kickv1alpha1.KickRequestPhaseExecuting)) {
		t.Fatal("Executing must not be notified by default")
	}
	if !phaseSelected([]kickv1alpha1.KickRequestPhase{kickv1alpha1.KickRequestPhaseExecuting}, string(kickv1alpha1.KickRequestPhaseExecuting)) {
		t.Fatal("an explicitly configured phase must be notified")
	}
}

func TestPolicyMatchesRespectsSuspendAndSelector(t *testing.T) {
	policy := &kickv1alpha1.NotificationPolicy{
		Spec: kickv1alpha1.NotificationPolicySpec{
			WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
		},
	}
	event := testEvent().WithWorkloadLabels(map[string]string{"app": "web"})

	matches, err := policyMatches(policy, event)
	if err != nil || !matches {
		t.Fatalf("expected a match, got %v (err %v)", matches, err)
	}

	matches, err = policyMatches(policy, testEvent().WithWorkloadLabels(map[string]string{"app": "api"}))
	if err != nil || matches {
		t.Fatalf("expected no match, got %v (err %v)", matches, err)
	}

	policy.Spec.Suspend = true
	matches, err = policyMatches(policy, event)
	if err != nil || matches {
		t.Fatalf("a suspended policy must not match, got %v (err %v)", matches, err)
	}
}

func TestDeliverPostsToWebhookWithBearerToken(t *testing.T) {
	var (
		mu         sync.Mutex
		gotAuth    string
		gotPayload []byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotPayload = buf
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	policy := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "alerts", Namespace: "prod"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{
				URL:            server.URL,
				TimeoutSeconds: 5,
				Auth:           &kickv1alpha1.NotificationAuth{BearerToken: &kickv1alpha1.SecretKeyRef{Name: "hook", Key: "token"}},
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "hook", Namespace: "prod"},
		Data:       map[string][]byte{"token": []byte("s3cr3t")},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).
		WithObjects(policy, secret).WithStatusSubresource(policy).Build()

	d := NewWebhookDispatcher(c, 4)
	d.deliver(context.Background(), testEvent())

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "Bearer s3cr3t" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if !strings.Contains(string(gotPayload), `"requestName":"web-1"`) {
		t.Fatalf("unexpected payload: %s", gotPayload)
	}
}

// A 4xx response is permanent; retrying it wastes a worker.
func TestSendDoesNotRetryClientErrors(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	policy := &kickv1alpha1.NotificationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "alerts", Namespace: "prod"},
		Spec: kickv1alpha1.NotificationPolicySpec{
			Webhook: kickv1alpha1.NotificationWebhook{URL: server.URL, TimeoutSeconds: 5},
		},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(policy).Build()

	d := NewWebhookDispatcher(c, 4)
	if err := d.send(context.Background(), policy, []byte("{}")); err == nil {
		t.Fatal("expected an error")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly one attempt, got %d", attempts)
	}
}

// Notify must never block reconciliation, even with a full queue.
func TestNotifyDropsOldestWhenQueueIsFull(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()
	d := NewWebhookDispatcher(c, 1)

	var dropped int
	d.dropped = func() { dropped++ }

	done := make(chan struct{})
	go func() {
		defer close(done)
		d.Notify(testEvent())
		d.Notify(testEvent())
		d.Notify(testEvent())
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Notify blocked on a full queue")
	}
	if dropped == 0 {
		t.Fatal("expected at least one dropped event")
	}
}

func TestNoopDispatcherIsSafe(t *testing.T) {
	var d Dispatcher = Noop{}
	d.Notify(testEvent())
}
