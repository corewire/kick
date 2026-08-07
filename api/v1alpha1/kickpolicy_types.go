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

// KickPolicyScheduleSource configures schedule/window source behavior.
type KickPolicyScheduleSource string

const (
	KickPolicyScheduleSourceProvider KickPolicyScheduleSource = "Provider"
	KickPolicyScheduleSourceNone     KickPolicyScheduleSource = "None"
)

// KickPolicyDiscoverySpec controls which workloads a policy manages and which of
// their dependency changes trigger a restart. Both selectors are optional; an
// empty or omitted selector matches everything on its axis.
type KickPolicyDiscoverySpec struct {
	// WorkloadSelector limits which workloads this policy manages (the actors that
	// may be restarted). Omit to manage all supported workloads in the namespace.
	// +optional
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
	// DependencySelector limits which consumed Secret/ConfigMap changes trigger a
	// restart. Omit to treat every discovered dependency as a trigger.
	// +optional
	DependencySelector *metav1.LabelSelector `json:"dependencySelector,omitempty"`
}

// KickPolicyWindowKind selects allow or deny semantics for a native window.
type KickPolicyWindowKind string

const (
	KickPolicyWindowKindAllow KickPolicyWindowKind = "Allow"
	KickPolicyWindowKindDeny  KickPolicyWindowKind = "Deny"
)

// KickPolicyWindow is a KICK-native execution window evaluated without a GitOps provider.
type KickPolicyWindow struct {
	// +kubebuilder:validation:Enum=Allow;Deny
	Kind KickPolicyWindowKind `json:"kind"`
	// Schedule is a standard 5-field cron expression marking each window start.
	Schedule string `json:"schedule"`
	// Duration is how long the window stays open from each start (e.g. "1h").
	// +kubebuilder:validation:Pattern=`^([0-9]+(ns|us|µs|ms|s|m|h))+$`
	Duration string `json:"duration"`
	// TimeZone is the IANA zone used to evaluate the schedule (default UTC).
	TimeZone string `json:"timeZone,omitempty"`
}

// KickPolicyScheduleSpec configures schedule source behavior.
type KickPolicyScheduleSpec struct {
	// +kubebuilder:default:=Provider
	// +kubebuilder:validation:Enum=Provider;None
	Source KickPolicyScheduleSource `json:"source,omitempty"`
	// Windows are KICK-native execution windows evaluated without a GitOps provider.
	Windows []KickPolicyWindow `json:"windows,omitempty"`
}

// KickPolicyGitOpsSpec configures provider gate behavior.
type KickPolicyGitOpsSpec struct {
	// Provider selects the GitOps gate. "None" (the default) restarts without
	// consulting any GitOps tool, gated only by any native schedule windows.
	// +kubebuilder:default:=None
	// +kubebuilder:validation:Enum=None;Auto;ArgoCD;Flux
	Provider KickPolicyProvider `json:"provider,omitempty"`
	// +kubebuilder:default:=true
	RequireReconciled *bool                  `json:"requireReconciled,omitempty"`
	Schedule          KickPolicyScheduleSpec `json:"schedule,omitempty"`
}

// KickPolicySpec configures workload scope and kick behavior.
type KickPolicySpec struct {
	Discovery KickPolicyDiscoverySpec `json:"discovery"`
	// +optional
	GitOps KickPolicyGitOpsSpec `json:"gitOps,omitempty"`
	// +kubebuilder:default:="30s"
	// +kubebuilder:validation:Pattern=`^$|^([0-9]+(ns|us|µs|ms|s|m|h))+$`
	MinInterval string `json:"minInterval,omitempty"`
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
