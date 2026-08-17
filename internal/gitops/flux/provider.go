package flux

import (
	"context"
	"fmt"
	"time"

	"github.com/corewire/kick/internal/gitops"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelKustomizationName      = "kustomize.toolkit.fluxcd.io/name"
	labelKustomizationNamespace = "kustomize.toolkit.fluxcd.io/namespace"
	labelHelmReleaseName        = "helm.toolkit.fluxcd.io/name"
	labelHelmReleaseNamespace   = "helm.toolkit.fluxcd.io/namespace"
)

var (
	KustomizationGVK = schema.GroupVersionKind{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization"}
	helmReleaseGVK   = schema.GroupVersionKind{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease"}
)

type Provider struct {
	Client client.Client
}

func (p *Provider) Name() string { return "flux" }

func (p *Provider) Detect(workload client.Object) gitops.DetectionResult {
	owner := ownerFromLabels(workload)
	if owner.Name == "" || owner.Namespace == "" {
		return gitops.DetectionResult{}
	}
	return gitops.DetectionResult{Confident: true, Message: "flux ownership labels present"}
}

func (p *Provider) ResolveOwner(_ context.Context, workload client.Object) (gitops.Owner, error) {
	owner := ownerFromLabels(workload)
	if owner.Name == "" || owner.Namespace == "" {
		return gitops.Owner{}, ResolutionError{Reason: gitops.GateOwnerUnknown, Message: "flux ownership labels missing"}
	}
	owner.Provider = p.Name()
	return owner, nil
}

func (p *Provider) EvaluateGate(ctx context.Context, owner gitops.Owner, _ time.Time) (gitops.GateDecision, error) {
	if owner.Name == "" || owner.Namespace == "" || owner.APIVersion == "" || owner.Kind == "" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOwnerUnknown, Message: "empty flux owner"}, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(owner.APIVersion)
	obj.SetKind(owner.Kind)
	if err := p.Client.Get(ctx, types.NamespacedName{Namespace: owner.Namespace, Name: owner.Name}, obj); err != nil {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateProviderUnavailable, Message: err.Error()}, nil
	}

	suspended, _, _ := unstructured.NestedBool(obj.Object, "spec", "suspend")
	if suspended {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOwnerOutOfSync, Message: "flux owner is suspended"}, nil
	}

	readyStatus, readyReason, readyMsg := conditionSummary(obj, "Ready")
	reconcilingStatus, _, _ := conditionSummary(obj, "Reconciling")
	if reconcilingStatus == "True" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: true, Reason: gitops.GateOwnerReconciling, Message: messageOrDefault(readyMsg, "flux owner is reconciling")}, nil
	}

	if readyStatus == "True" {
		return gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: messageOrDefault(readyMsg, "flux owner ready")}, nil
	}
	if readyStatus == "False" {
		reason := gitops.GateOwnerOutOfSync
		if readyReason == "Progressing" {
			reason = gitops.GateOwnerReconciling
		}
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: reason == gitops.GateOwnerReconciling, Reason: reason, Message: messageOrDefault(readyMsg, "flux owner not ready")}, nil
	}

	return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOwnerUnknown, Message: "flux owner status has no Ready condition"}, nil
}

type ResolutionError struct {
	Reason  gitops.GateReason
	Message string
}

func (e ResolutionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}

// GateReason implements gitops.GateReasoner.
func (e ResolutionError) GateReason() gitops.GateReason {
	return e.Reason
}

func ownerFromLabels(workload client.Object) gitops.Owner {
	if workload == nil {
		return gitops.Owner{}
	}
	labels := workload.GetLabels()
	if labels == nil {
		return gitops.Owner{}
	}
	if name := labels[labelKustomizationName]; name != "" {
		namespace := labels[labelKustomizationNamespace]
		if namespace == "" {
			namespace = workload.GetNamespace()
		}
		return gitops.Owner{APIVersion: KustomizationGVK.GroupVersion().String(), Kind: KustomizationGVK.Kind, Namespace: namespace, Name: name}
	}
	if name := labels[labelHelmReleaseName]; name != "" {
		namespace := labels[labelHelmReleaseNamespace]
		if namespace == "" {
			namespace = workload.GetNamespace()
		}
		return gitops.Owner{APIVersion: helmReleaseGVK.GroupVersion().String(), Kind: helmReleaseGVK.Kind, Namespace: namespace, Name: name}
	}
	return gitops.Owner{}
}

func conditionSummary(obj *unstructured.Unstructured, condType string) (status, reason, message string) {
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return "", "", ""
	}
	for _, c := range conditions {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		t, _, _ := unstructured.NestedString(m, "type")
		if t != condType {
			continue
		}
		status, _, _ = unstructured.NestedString(m, "status")
		reason, _, _ = unstructured.NestedString(m, "reason")
		message, _, _ = unstructured.NestedString(m, "message")
		return status, reason, message
	}
	return "", "", ""
}

func messageOrDefault(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
