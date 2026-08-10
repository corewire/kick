package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KickRequestPhase is the lifecycle state of a durable restart request.
type KickRequestPhase string

const (
	KickRequestPhasePending           KickRequestPhase = "Pending"
	KickRequestPhaseWaitingForGate    KickRequestPhase = "WaitingForGate"
	KickRequestPhaseWaitingForOwner   KickRequestPhase = "WaitingForOwner"
	KickRequestPhaseWaitingForAppSync KickRequestPhase = "WaitingForApplicationSync"
	KickRequestPhaseWaitingForRollout KickRequestPhase = "WaitingForRollout"
	KickRequestPhaseExecuting         KickRequestPhase = "Executing"
	KickRequestPhaseSucceeded         KickRequestPhase = "Succeeded"
	KickRequestPhaseNoLongerRequired  KickRequestPhase = "NoLongerRequired"
	KickRequestPhaseFailed            KickRequestPhase = "Failed"
	// KickRequestPhaseDryRun is terminal: the restart was required and allowed,
	// but the policy runs in dry-run mode so no workload was patched.
	KickRequestPhaseDryRun KickRequestPhase = "DryRun"
)

// ObjectReference identifies the workload KICK may restart.
type ObjectReference struct {
	// +kubebuilder:default:=apps/v1
	APIVersion string `json:"apiVersion"`
	// +kubebuilder:default:=Deployment
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet;Rollout
	Kind string `json:"kind"`
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// GitOpsOwnerStatus records the provider owner resolved from live cluster state.
type GitOpsOwnerStatus struct {
	Provider   string `json:"provider,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name,omitempty"`
	Project    string `json:"project,omitempty"`
}

// GateStatus records the last provider gate decision. RequeueAt is advisory.
type GateStatus struct {
	Reason    string       `json:"reason,omitempty"`
	Message   string       `json:"message,omitempty"`
	RequeueAt *metav1.Time `json:"requeueAt,omitempty"`
}

// RolloutStatus records the rollout used for the last freshness comparison.
type RolloutStatus struct {
	ReplicaSet string       `json:"replicaSet,omitempty"`
	StartedAt  *metav1.Time `json:"startedAt,omitempty"`
}

// PolicyReference records the KickPolicy that created a request (audit).
type PolicyReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// KickRequestSpec describes the workload to reevaluate.
type KickRequestSpec struct {
	TargetRef ObjectReference `json:"targetRef"`
	// PolicyRef records the owning KickPolicy for audit and reverse lookup.
	// +optional
	PolicyRef *PolicyReference `json:"policyRef,omitempty"`
}

// KickRequestStatus is diagnostic and audit state. Live state remains authoritative.
type KickRequestStatus struct {
	// +kubebuilder:validation:Enum=Pending;WaitingForGate;WaitingForOwner;WaitingForApplicationSync;WaitingForRollout;Executing;Succeeded;NoLongerRequired;Failed;DryRun
	Phase                          KickRequestPhase   `json:"phase,omitempty"`
	Owner                          GitOpsOwnerStatus  `json:"owner,omitempty"`
	Gate                           GateStatus         `json:"gate,omitempty"`
	LatestObservedDependencyChange *metav1.Time       `json:"latestObservedDependencyChange,omitempty"`
	CurrentRollout                 RolloutStatus      `json:"currentRollout,omitempty"`
	Conditions                     []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kick
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetRef.name`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.status.owner.provider`
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.status.owner.name`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.gate.reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type KickRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KickRequestSpec   `json:"spec,omitempty"`
	Status KickRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KickRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KickRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KickRequest{}, &KickRequestList{})
}
