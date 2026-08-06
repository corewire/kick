package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectReference identifies the workload KICK may restart.
type ObjectReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
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

// KickRequestSpec describes the workload to reevaluate.
type KickRequestSpec struct {
	TargetRef ObjectReference `json:"targetRef"`
}

// KickRequestStatus is diagnostic and audit state. Live state remains authoritative.
type KickRequestStatus struct {
	Phase                          string             `json:"phase,omitempty"`
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
