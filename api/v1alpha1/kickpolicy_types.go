package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KickPolicyDiscoveryMode configures dependency/workload discovery behavior.
type KickPolicyDiscoveryMode string

const (
	KickPolicyDiscoveryModeAuto KickPolicyDiscoveryMode = "Auto"
)

// KickPolicyProvider configures GitOps ownership resolution behavior.
type KickPolicyProvider string

const (
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

// KickPolicyDiscoverySpec controls workload and dependency discovery.
type KickPolicyDiscoverySpec struct {
	// +kubebuilder:validation:Enum=Auto
	Mode             KickPolicyDiscoveryMode `json:"mode"`
	WorkloadSelector *metav1.LabelSelector   `json:"workloadSelector,omitempty"`
}

// KickPolicyScheduleSpec configures schedule source behavior.
type KickPolicyScheduleSpec struct {
	// +kubebuilder:default:=Provider
	// +kubebuilder:validation:Enum=Provider;None
	Source KickPolicyScheduleSource `json:"source,omitempty"`
}

// KickPolicyGitOpsSpec configures provider gate behavior.
type KickPolicyGitOpsSpec struct {
	// +kubebuilder:validation:Enum=Auto;ArgoCD;Flux
	Provider KickPolicyProvider `json:"provider"`
	// +kubebuilder:default:=true
	RequireReconciled *bool                  `json:"requireReconciled,omitempty"`
	Schedule          KickPolicyScheduleSpec `json:"schedule,omitempty"`
}

// KickPolicySpec configures workload scope and kick behavior.
type KickPolicySpec struct {
	Discovery KickPolicyDiscoverySpec `json:"discovery"`
	GitOps    KickPolicyGitOpsSpec    `json:"gitOps"`
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
