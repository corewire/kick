// Package kargo gates restarts on Kargo promotion state.
//
// Kargo never writes to workloads directly: it promotes Freight by updating a
// Git repository, and an Argo CD Application then applies the result. Workload
// ownership is therefore resolved by Argo CD, and the Kargo Stage is derived
// from the Application's "kargo.akuity.io/authorized-stage" annotation.
//
// The Kargo gate is strictly stricter than the Argo CD gate: an Application can
// be Synced and Healthy in the middle of a multi-step Promotion, so restarting
// then would race the promotion.
package kargo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/corewire/kick/internal/gitops"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// authorizedStageAnnotation lists the Stages allowed to manage a resource as a
// comma-separated list of "<project>:<stage>" entries.
const authorizedStageAnnotation = "kargo.akuity.io/authorized-stage"

var (
	// StageGVK identifies the Kargo Stage CRD.
	StageGVK         = schema.GroupVersionKind{Group: "kargo.akuity.io", Version: "v1alpha1", Kind: "Stage"}
	promotionListGVK = schema.GroupVersionKind{Group: "kargo.akuity.io", Version: "v1alpha1", Kind: "PromotionList"}
)

// terminalPromotionPhases are the Kargo promotion phases that no longer change
// cluster state.
var terminalPromotionPhases = map[string]struct{}{
	"Succeeded": {},
	"Failed":    {},
	"Errored":   {},
	"Aborted":   {},
}

// ApplicationResolver resolves the Argo CD Application owning a workload and
// evaluates its gate. The Argo CD provider implements it.
type ApplicationResolver interface {
	ResolveOwner(context.Context, client.Object) (gitops.Owner, error)
	EvaluateGate(context.Context, gitops.Owner, time.Time) (gitops.GateDecision, error)
}

type Provider struct {
	Client client.Client
	// ArgoCD resolves the owning Application and evaluates the underlying Argo CD
	// gate. Kargo alone cannot answer either question.
	ArgoCD ApplicationResolver
}

func (p *Provider) Name() string { return "kargo" }

// Detect always returns a non-confident result. A Kargo Stage is not discoverable
// from workload metadata - only the owning Argo CD Application carries the
// authorization annotation - so a policy must select Kargo explicitly. Auto
// detection keeps resolving such workloads to the Argo CD provider.
func (p *Provider) Detect(client.Object) gitops.DetectionResult {
	return gitops.DetectionResult{}
}

// ResolveOwner returns the Kargo Stage governing the workload. The Argo CD
// Application is kept in Owner.Project so EvaluateGate can re-check it without
// touching the workload again.
func (p *Provider) ResolveOwner(ctx context.Context, workload client.Object) (gitops.Owner, error) {
	if p.ArgoCD == nil {
		return gitops.Owner{}, ResolutionError{Reason: gitops.GateProviderUnavailable, Message: "argocd resolver not configured"}
	}
	appOwner, err := p.ArgoCD.ResolveOwner(ctx, workload)
	if err != nil {
		return gitops.Owner{}, err
	}
	app, err := p.getApplication(ctx, types.NamespacedName{Namespace: appOwner.Namespace, Name: appOwner.Name})
	if err != nil {
		return gitops.Owner{}, ResolutionError{Reason: gitops.GateProviderUnavailable, Message: err.Error()}
	}
	raw := app.GetAnnotations()[authorizedStageAnnotation]
	stage, ok := singleAuthorizedStage(raw)
	if !ok {
		if strings.Contains(raw, ",") {
			return gitops.Owner{}, ResolutionError{Reason: gitops.GateAmbiguousOwner, Message: "application authorizes more than one kargo stage"}
		}
		return gitops.Owner{}, ResolutionError{Reason: gitops.GateOwnerUnknown, Message: "application has no kargo authorized-stage annotation"}
	}
	return gitops.Owner{
		Provider:   p.Name(),
		APIVersion: StageGVK.GroupVersion().String(),
		Kind:       StageGVK.Kind,
		Namespace:  stage.project,
		Name:       stage.name,
		// Project carries the owning Application as "<namespace>/<name>".
		Project: appOwner.Namespace + "/" + appOwner.Name,
	}, nil
}

// EvaluateGate blocks while the Stage has an in-flight Promotion and otherwise
// falls through to the Argo CD gate for the Application that actually applies
// the manifests.
func (p *Provider) EvaluateGate(ctx context.Context, owner gitops.Owner, now time.Time) (gitops.GateDecision, error) {
	if owner.Name == "" || owner.Namespace == "" {
		return blocked(gitops.GateOwnerUnknown, "empty kargo owner"), nil
	}

	stage := &unstructured.Unstructured{}
	stage.SetGroupVersionKind(StageGVK)
	if err := p.Client.Get(ctx, types.NamespacedName{Namespace: owner.Namespace, Name: owner.Name}, stage); err != nil {
		return blocked(gitops.GateProviderUnavailable, err.Error()), nil
	}

	// status.currentPromotion is set for the whole duration of a promotion,
	// including the phases where Kargo has already written to Git but Argo CD has
	// not observed it yet.
	if name, found, _ := unstructured.NestedString(stage.Object, "status", "currentPromotion", "name"); found && name != "" {
		return reconciling("kargo stage has a promotion in progress"), nil
	}

	inFlight, err := p.hasInFlightPromotion(ctx, owner.Namespace, owner.Name)
	if err != nil {
		return blocked(gitops.GateProviderUnavailable, err.Error()), nil
	}
	if inFlight {
		return reconciling("kargo promotion is pending or running"), nil
	}

	appNamespace, appName, ok := strings.Cut(owner.Project, "/")
	if !ok {
		return blocked(gitops.GateOwnerUnknown, "kargo owner does not reference an argocd application"), nil
	}
	if p.ArgoCD == nil {
		return blocked(gitops.GateProviderUnavailable, "argocd resolver not configured"), nil
	}
	return p.ArgoCD.EvaluateGate(ctx, gitops.Owner{
		Provider:   "argocd",
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
		Namespace:  appNamespace,
		Name:       appName,
	}, now)
}

// hasInFlightPromotion reports whether a non-terminal Promotion targets the
// Stage. Promotions are matched on spec.stage rather than the
// kargo.akuity.io/stage label because that label may be hash-shortened.
func (p *Provider) hasInFlightPromotion(ctx context.Context, namespace, stage string) (bool, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(promotionListGVK)
	if err := p.Client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	for i := range list.Items {
		item := &list.Items[i]
		target, _, _ := unstructured.NestedString(item.Object, "spec", "stage")
		if target != stage {
			continue
		}
		phase, _, _ := unstructured.NestedString(item.Object, "status", "phase")
		// An empty phase means the promotion controller has not written status
		// yet, which is the earliest point of an in-flight promotion.
		if _, terminal := terminalPromotionPhases[phase]; !terminal {
			return true, nil
		}
	}
	return false, nil
}

func (p *Provider) getApplication(ctx context.Context, key types.NamespacedName) (*unstructured.Unstructured, error) {
	app := &unstructured.Unstructured{}
	app.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"})
	if err := p.Client.Get(ctx, key, app); err != nil {
		return nil, err
	}
	return app, nil
}

type stageRef struct {
	project string
	name    string
}

// singleAuthorizedStage parses the annotation value and only accepts exactly
// one "<project>:<stage>" entry. Multiple authorized Stages make ownership
// ambiguous, which must block a restart.
func singleAuthorizedStage(raw string) (stageRef, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.Contains(trimmed, ",") {
		return stageRef{}, false
	}
	project, name, ok := strings.Cut(trimmed, ":")
	if !ok || project == "" || name == "" {
		return stageRef{}, false
	}
	return stageRef{project: project, name: name}, true
}

func blocked(reason gitops.GateReason, message string) gitops.GateDecision {
	return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: reason, Message: message}
}

func reconciling(message string) gitops.GateDecision {
	return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: true, Reason: gitops.GateOwnerReconciling, Message: message}
}

type ResolutionError struct {
	Reason  gitops.GateReason
	Message string
}

func (e ResolutionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}
