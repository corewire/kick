package kargo

import (
	"context"
	"encoding/json"
	"fmt"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/gitops"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reverifyAnnotation is the Kargo v1.11.4 request to rerun verification.
// KICK writes VerificationRequest JSON, {"id":"<verification id>"}.
const reverifyAnnotation = "kargo.akuity.io/reverify"

// terminalVerificationPhases match Kargo VerificationPhase.IsTerminal.
// Promotion phases use Succeeded; verification phases use Successful.
var terminalVerificationPhases = map[string]struct{}{
	"Successful":   {},
	"Failed":       {},
	"Error":        {},
	"Aborted":      {},
	"Inconclusive": {},
}

// verificationView is the current Freight verification state KICK can read.
// Freight history can still advance between this read and the restart patch.
// Those writes are not atomic.
type verificationView struct {
	active          bool
	message         string
	collectionID    string
	verificationID  string
	hasVerification bool
}

// ReverifyResult is the outcome of one reverification attempt.
// Retry means the annotation was not sent and the caller should try later
// without issuing another restart.
type ReverifyResult struct {
	Status kickv1alpha1.KargoReverificationStatus
	Retry  bool
}

// SnapshotVerification stores the Stage and current verification identity.
// An empty verification ID is Skipped: Kargo refuses reverification without it.
func (p *Provider) SnapshotVerification(ctx context.Context, namespace, name string) (kickv1alpha1.KargoReverificationStatus, error) {
	stage, err := p.getStage(ctx, namespace, name)
	if err != nil {
		return kickv1alpha1.KargoReverificationStatus{}, err
	}
	view, err := readVerification(stage)
	if err != nil {
		return kickv1alpha1.KargoReverificationStatus{}, err
	}
	status := kickv1alpha1.KargoReverificationStatus{
		StageNamespace:      namespace,
		StageName:           name,
		FreightCollectionID: view.collectionID,
		VerificationID:      view.verificationID,
		State:               kickv1alpha1.KargoReverificationPending,
	}
	if view.verificationID == "" {
		status.State = kickv1alpha1.KargoReverificationSkipped
	}
	return status, nil
}

// RequestReverification patches kargo.akuity.io/reverify once the stored
// identity still matches and Kargo is idle. It never creates a Promotion.
func (p *Provider) RequestReverification(ctx context.Context, recorded kickv1alpha1.KargoReverificationStatus) (ReverifyResult, error) {
	result := ReverifyResult{Status: recorded}
	if recorded.VerificationID == "" || recorded.StageNamespace == "" || recorded.StageName == "" {
		result.Status.State = kickv1alpha1.KargoReverificationSkipped
		return result, nil
	}
	stage, err := p.getStage(ctx, recorded.StageNamespace, recorded.StageName)
	if err != nil {
		return result, err
	}
	view, err := readVerification(stage)
	if err != nil {
		return result, err
	}
	active, err := p.promotionActive(ctx, recorded.StageNamespace, recorded.StageName, stage)
	if err != nil {
		return result, err
	}
	if view.active || active {
		result.Retry = true
		return result, nil
	}
	if view.collectionID != recorded.FreightCollectionID {
		result.Status.State = kickv1alpha1.KargoReverificationSkipped
		return result, nil
	}
	if view.verificationID != recorded.VerificationID {
		result.Status.State = kickv1alpha1.KargoReverificationRequested
		return result, nil
	}
	if other, ok := annotationVerificationID(stage); ok && other != recorded.VerificationID {
		result.Retry = true
		return result, nil
	}
	if annotationHasID(stage, recorded.VerificationID) {
		result.Status.State = kickv1alpha1.KargoReverificationRequested
		return result, nil
	}
	if err := p.patchReverify(ctx, stage, recorded.VerificationID); err != nil {
		return result, err
	}
	result.Status.State = kickv1alpha1.KargoReverificationRequested
	return result, nil
}

func (p *Provider) getStage(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	stage := &unstructured.Unstructured{}
	stage.SetGroupVersionKind(StageGVK)
	if err := p.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, stage); err != nil {
		return nil, err
	}
	return stage, nil
}

func (p *Provider) promotionActive(ctx context.Context, namespace, stageName string, stage *unstructured.Unstructured) (bool, error) {
	name, found, err := unstructured.NestedString(stage.Object, "status", "currentPromotion", "name")
	if err != nil {
		return false, err
	}
	if found && name != "" {
		return true, nil
	}
	return p.hasInFlightPromotion(ctx, namespace, stageName)
}

func (p *Provider) patchReverify(ctx context.Context, stage *unstructured.Unstructured, verificationID string) error {
	original := stage.DeepCopy()
	annotations := stage.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[reverifyAnnotation] = reverifyAnnotationValue(verificationID)
	stage.SetAnnotations(annotations)
	return p.Client.Patch(ctx, stage, client.MergeFrom(original))
}

func reverifyAnnotationValue(verificationID string) string {
	encoded, err := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: verificationID})
	if err != nil {
		return ""
	}
	return string(encoded)
}

func annotationHasID(stage *unstructured.Unstructured, verificationID string) bool {
	id, ok := annotationVerificationID(stage)
	return ok && id == verificationID
}

func annotationVerificationID(stage *unstructured.Unstructured) (string, bool) {
	raw := stage.GetAnnotations()[reverifyAnnotation]
	if raw == "" {
		return "", false
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil && parsed.ID != "" {
		return parsed.ID, true
	}
	return raw, true
}

// verificationGate blocks while verification is active. Terminal outcomes and
// status.health do not block. Stages without spec.verification keep the
// promotion-only gate.
func verificationGate(stage *unstructured.Unstructured) (gitops.GateDecision, bool, error) {
	view, err := readVerification(stage)
	if err != nil {
		return gitops.GateDecision{}, false, err
	}
	if !view.active {
		return gitops.GateDecision{}, false, nil
	}
	return reconciling(view.message), true, nil
}

func readVerification(stage *unstructured.Unstructured) (verificationView, error) {
	view := verificationView{}
	configured, err := verificationConfigured(stage)
	if err != nil {
		return view, err
	}

	history, found, err := unstructured.NestedSlice(stage.Object, "status", "freightHistory")
	if err != nil {
		return view, fmt.Errorf("unreadable freight history: %w", err)
	}
	if !found || len(history) == 0 {
		return view, nil
	}
	collection, ok := history[0].(map[string]any)
	if !ok {
		return view, fmt.Errorf("unreadable freight history: current entry is not an object")
	}
	view.collectionID, _, err = unstructured.NestedString(collection, "id")
	if err != nil {
		return view, fmt.Errorf("unreadable freight history: %w", err)
	}

	records, recordsFound, err := unstructured.NestedSlice(collection, "verificationHistory")
	if err != nil {
		return view, fmt.Errorf("unreadable verification history: %w", err)
	}
	if recordsFound {
		for i, raw := range records {
			entry, ok := raw.(map[string]any)
			if !ok {
				return view, fmt.Errorf("unreadable verification history: entry %d is not an object", i)
			}
			phase, _, err := unstructured.NestedString(entry, "phase")
			if err != nil {
				return view, fmt.Errorf("unreadable verification history: %w", err)
			}
			if _, terminal := terminalVerificationPhases[phase]; !terminal {
				view.active = true
				view.message = "kargo verification is pending or running"
			}
			if i == 0 {
				view.hasVerification = true
				view.verificationID, _, err = unstructured.NestedString(entry, "id")
				if err != nil {
					return view, fmt.Errorf("unreadable verification history: %w", err)
				}
			}
		}
	}
	if view.active {
		return view, nil
	}
	if !configured || view.hasVerification {
		return view, nil
	}
	matched, err := lastPromotionMatchesCollection(stage, collection)
	if err != nil {
		return view, err
	}
	if matched {
		view.active = true
		view.message = "kargo verification for the current freight has not been recorded"
	}
	return view, nil
}

func verificationConfigured(stage *unstructured.Unstructured) (bool, error) {
	raw, found, err := unstructured.NestedFieldNoCopy(stage.Object, "spec", "verification")
	if err != nil || !found || raw == nil {
		return false, err
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return false, fmt.Errorf("unreadable verification config: spec.verification is not an object")
	}
	return len(object) > 0, nil
}

func lastPromotionMatchesCollection(stage *unstructured.Unstructured, collection map[string]any) (bool, error) {
	phase, found, err := unstructured.NestedString(stage.Object, "status", "lastPromotion", "status", "phase")
	if err != nil {
		return false, fmt.Errorf("unreadable last promotion: %w", err)
	}
	if !found || phase != "Succeeded" {
		return false, nil
	}
	freightName, _, err := unstructured.NestedString(stage.Object, "status", "lastPromotion", "freight", "name")
	if err != nil {
		return false, fmt.Errorf("unreadable last promotion: %w", err)
	}
	return freightCollectionIncludes(collection, freightName)
}

func freightCollectionIncludes(collection map[string]any, freightName string) (bool, error) {
	if freightName == "" || collection == nil {
		return false, nil
	}
	items, found, err := unstructured.NestedMap(collection, "items")
	if err != nil {
		return false, fmt.Errorf("unreadable freight collection: %w", err)
	}
	if !found {
		return false, nil
	}
	for _, raw := range items {
		ref, ok := raw.(map[string]any)
		if !ok {
			return false, fmt.Errorf("unreadable freight collection: item is not an object")
		}
		name, _, err := unstructured.NestedString(ref, "name")
		if err != nil {
			return false, err
		}
		if name == freightName {
			return true, nil
		}
	}
	return false, nil
}
