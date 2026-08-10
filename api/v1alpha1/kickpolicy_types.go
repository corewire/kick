package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KickPolicyProvider configures GitOps ownership resolution behavior.
type KickPolicyProvider string

const (
	KickPolicyProviderNone   KickPolicyProvider = "None"
	KickPolicyProviderAuto   KickPolicyProvider = "Auto"
	KickPolicyProviderArgoCD KickPolicyProvider = "ArgoCD"
	KickPolicyProviderFlux   KickPolicyProvider = "Flux"
)

// KickPolicyDiscoverySpec controls which workloads a policy manages and which of
// their dependency changes trigger a restart.
type KickPolicyDiscoverySpec struct {
	// WorkloadSelector limits which workloads this policy manages (the actors that
	// may be restarted). It is required; an explicit empty selector ({}) opts in
	// to every supported workload in the namespace, so wide blast radius is never
	// accidental.
	// +kubebuilder:validation:Required
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector"`
	// DependencySelector limits which consumed Secret/ConfigMap changes trigger a
	// restart. Omit to treat every discovered dependency as a trigger.
	// +optional
	DependencySelector *metav1.LabelSelector `json:"dependencySelector,omitempty"`
}

// KickPolicyWindowType selects allow or deny semantics for a native window.
type KickPolicyWindowType string

const (
	KickPolicyWindowTypeAllow KickPolicyWindowType = "Allow"
	KickPolicyWindowTypeDeny  KickPolicyWindowType = "Deny"
)

// KickPolicyWindow is a KICK-native restart window, evaluated independently of
// any GitOps provider.
type KickPolicyWindow struct {
	// +kubebuilder:validation:Enum=Allow;Deny
	Type KickPolicyWindowType `json:"type"`
	// Cron is a standard 5-field cron expression marking each window start.
	Cron string `json:"cron"`
	// Duration is how long the window stays open from each start (e.g. "1h").
	// +kubebuilder:validation:Pattern=`^([0-9]+(ns|us|µs|ms|s|m|h))+$`
	Duration string `json:"duration"`
	// TimeZone is the IANA zone used to evaluate the cron expression (default UTC).
	TimeZone string `json:"timeZone,omitempty"`
}

// KickPolicyScheduleSpec is the native time gate: pure scheduling, no GitOps.
type KickPolicyScheduleSpec struct {
	// Windows are KICK-native restart windows. Omit to allow restarts at any time.
	// +optional
	Windows []KickPolicyWindow `json:"windows,omitempty"`
}

// KickPolicyGitOpsSpec configures provider gate behavior.
type KickPolicyGitOpsSpec struct {
	// Provider selects the GitOps gate. "None" (the default) restarts without
	// consulting any GitOps tool, gated only by any native schedule windows.
	// +kubebuilder:default:=None
	// +kubebuilder:validation:Enum=None;Auto;ArgoCD;Flux
	Provider KickPolicyProvider `json:"provider,omitempty"`
	// RequireReconciled is a correctness gate (not a schedule): wait until the
	// owning application has finished applying before restarting.
	// +kubebuilder:default:=true
	RequireReconciled *bool `json:"requireReconciled,omitempty"`
}

// KickPolicyRestartSpec groups restart behavior so future knobs are additive.
type KickPolicyRestartSpec struct {
	// MinInterval is the minimum time between restarts of the same workload.
	// +kubebuilder:default:="30s"
	// +kubebuilder:validation:Pattern=`^$|^([0-9]+(ns|us|µs|ms|s|m|h))+$`
	MinInterval string `json:"minInterval,omitempty"`
}

// KickPolicySpec configures workload scope and kick behavior.
type KickPolicySpec struct {
	// Suspend pauses the policy: it matches nothing and issues no restarts until
	// unset, without deleting the object.
	// +optional
	Suspend   bool                    `json:"suspend,omitempty"`
	Discovery KickPolicyDiscoverySpec `json:"discovery"`
	// +optional
	Schedule KickPolicyScheduleSpec `json:"schedule,omitempty"`
	// +optional
	GitOps KickPolicyGitOpsSpec `json:"gitOps,omitempty"`
	// +optional
	Restart KickPolicyRestartSpec `json:"restart,omitempty"`
}

// KickPolicyStatus reports policy matching and readiness state.
type KickPolicyStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	MatchedWorkloads   int32              `json:"matchedWorkloads,omitempty"`
	BlockedWorkloads   int32              `json:"blockedWorkloads,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kickpolicy
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Matched",type=integer,JSONPath=`.status.matchedWorkloads`
// +kubebuilder:printcolumn:name="Blocked",type=integer,JSONPath=`.status.blockedWorkloads`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type KickPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KickPolicySpec   `json:"spec,omitempty"`
	Status KickPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KickPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KickPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KickPolicy{}, &KickPolicyList{})
}
