package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretKeyRef selects a single key from a Secret in the same namespace as the
// NotificationPolicy. Credentials are never inlined in the spec.
type SecretKeyRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

// NotificationHeader adds a request header. Use ValueFrom for anything secret;
// Value is only for non-sensitive routing headers.
type NotificationHeader struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Value string `json:"value,omitempty"`
	// +optional
	ValueFrom *SecretKeyRef `json:"valueFrom,omitempty"`
}

// NotificationBasicAuth holds HTTP basic auth credential references.
type NotificationBasicAuth struct {
	Username SecretKeyRef `json:"username"`
	Password SecretKeyRef `json:"password"`
}

// NotificationAuth selects one authentication scheme. At most one field may be
// set.
type NotificationAuth struct {
	// BearerToken sends "Authorization: Bearer <token>".
	// +optional
	BearerToken *SecretKeyRef `json:"bearerToken,omitempty"`
	// Basic sends "Authorization: Basic <base64(user:pass)>".
	// +optional
	Basic *NotificationBasicAuth `json:"basic,omitempty"`
}

// NotificationTLS configures transport security for the webhook endpoint.
type NotificationTLS struct {
	// CABundle references a PEM bundle used to verify the endpoint certificate.
	// +optional
	CABundle *SecretKeyRef `json:"caBundle,omitempty"`
	// ClientCertificate references a Secret of type kubernetes.io/tls used for
	// mTLS. Both tls.crt and tls.key must be present.
	// +optional
	ClientCertificate *SecretKeyRef `json:"clientCertificate,omitempty"`
}

// NotificationWebhook is the delivery target.
type NotificationWebhook struct {
	// URL must be https unless the endpoint is in-cluster. Credentials must not
	// be embedded in the URL.
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`
	// +kubebuilder:default:=POST
	// +kubebuilder:validation:Enum=POST;PUT
	Method string `json:"method,omitempty"`
	// +optional
	Headers []NotificationHeader `json:"headers,omitempty"`
	// +optional
	Auth *NotificationAuth `json:"auth,omitempty"`
	// +optional
	TLS *NotificationTLS `json:"tls,omitempty"`
	// TimeoutSeconds bounds a single delivery attempt.
	// +kubebuilder:default:=10
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=120
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
}

// NotificationPolicySpec selects which KickRequest transitions are delivered
// where.
type NotificationPolicySpec struct {
	// Suspend stops delivery without deleting the object.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
	// Phases limits notifications to these KickRequest phases. Omit to notify on
	// every terminal phase (Succeeded, Failed, NoLongerRequired, DryRun).
	// +optional
	Phases []KickRequestPhase `json:"phases,omitempty"`
	// WorkloadSelector limits notifications to KickRequests whose target workload
	// carries matching labels. Omit to match every request in the namespace.
	// +optional
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
	Webhook          NotificationWebhook   `json:"webhook"`
}

// NotificationPolicyStatus reports delivery health. Counters are best-effort and
// reset when the operator restarts.
type NotificationPolicyStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	LastDeliveryTime *metav1.Time `json:"lastDeliveryTime,omitempty"`
	// LastError is the failure message of the most recent failed delivery. It
	// never contains request or response bodies.
	// +optional
	LastError  string             `json:"lastError,omitempty"`
	Delivered  int64              `json:"delivered,omitempty"`
	Failed     int64              `json:"failed,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=notificationpolicy
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.webhook.url`
// +kubebuilder:printcolumn:name="Delivered",type=integer,JSONPath=`.status.delivered`
// +kubebuilder:printcolumn:name="Failed",type=integer,JSONPath=`.status.failed`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type NotificationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationPolicySpec   `json:"spec,omitempty"`
	Status NotificationPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type NotificationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NotificationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NotificationPolicy{}, &NotificationPolicyList{})
}
