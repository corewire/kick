package argocd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/corewire/kick/internal/gitops"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	trackingIDAnnotation = "argocd.argoproj.io/tracking-id"
)

var (
	applicationGVK = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"}
	appProjectGVK  = schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "AppProject"}
)

type Provider struct {
	Client                client.Client
	ApplicationNamespaces []string
	ControlPlaneNamespace string
}

func (p *Provider) Name() string { return "argocd" }

func (p *Provider) Detect(workload client.Object) gitops.DetectionResult {
	if workload == nil {
		return gitops.DetectionResult{}
	}
	if workload.GetAnnotations()[trackingIDAnnotation] != "" {
		return gitops.DetectionResult{Confident: true, Message: "tracking-id annotation present"}
	}
	return gitops.DetectionResult{}
}

func (p *Provider) ResolveOwner(ctx context.Context, workload client.Object) (gitops.Owner, error) {
	owner, reason, err := p.resolveOwnerWithReason(ctx, workload)
	if err != nil {
		return gitops.Owner{}, err
	}
	if reason != gitops.GateAllowed {
		return gitops.Owner{}, ResolutionError{Reason: reason, Message: "owner not uniquely resolvable"}
	}
	return owner, nil
}

func (p *Provider) EvaluateGate(ctx context.Context, owner gitops.Owner, _ time.Time) (gitops.GateDecision, error) {
	if owner.Name == "" || owner.Namespace == "" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOwnerUnknown, Message: "empty owner"}, nil
	}

	app, err := p.getApplication(ctx, types.NamespacedName{Namespace: owner.Namespace, Name: owner.Name})
	if err != nil {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateProviderUnavailable, Message: err.Error()}, nil
	}

	project, _, _ := unstructured.NestedString(app.Object, "spec", "project")
	if project == "" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateProjectUnknown, Message: "missing application project"}, nil
	}
	projectObj, err := p.getAppProjectObject(ctx, project)
	if err != nil {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateProjectUnknown, Message: err.Error()}, nil
	}
	windows, _, _ := unstructured.NestedSlice(projectObj.Object, "spec", "syncWindows")
	windowMaps := make([]map[string]interface{}, 0, len(windows))
	for _, w := range windows {
		m, ok := w.(map[string]interface{})
		if ok {
			windowMaps = append(windowMaps, m)
		}
	}
	destinationNS, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
	destinationName, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "name")
	destinationServer, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "server")
	windowDecision, evalErr := evaluateSyncWindows(time.Now().UTC(), appWindowContext{Name: owner.Name, DestinationNS: destinationNS, DestinationName: destinationName, DestinationServer: destinationServer}, windowMaps)
	if evalErr != nil {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateConfigurationError, Message: evalErr.Error()}, nil
	}
	if !windowDecision.Allowed {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, RequeueAt: windowDecision.RequeueAt, Reason: gitops.GateOutsideSchedule, Message: "blocked by sync window"}, nil
	}

	if phase, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "phase"); phase != "" && phase != "Succeeded" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: true, Reason: gitops.GateOwnerReconciling, Message: "application operation in progress"}, nil
	}

	if syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status"); syncStatus != "Synced" {
		return gitops.GateDecision{Allowed: false, Reconciled: false, Reconciling: false, Reason: gitops.GateOwnerOutOfSync, Message: "application is not synced"}, nil
	}

	return gitops.GateDecision{Allowed: true, Reconciled: true, Reconciling: false, Reason: gitops.GateAllowed, Message: "application synced and project found"}, nil
}

type ResolutionError struct {
	Reason  gitops.GateReason
	Message string
}

func (e ResolutionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Reason, e.Message)
}

type trackingIdentity struct {
	AppName   string
	Group     string
	Kind      string
	Namespace string
	Name      string
}

func parseTrackingID(raw string) (trackingIdentity, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return trackingIdentity{}, errors.New("invalid tracking-id format")
	}
	appName := parts[0]
	gk := strings.Split(parts[1], "/")
	nn := strings.Split(parts[2], "/")
	if appName == "" || len(gk) != 2 || len(nn) != 2 || gk[1] == "" || nn[0] == "" || nn[1] == "" {
		return trackingIdentity{}, errors.New("invalid tracking-id components")
	}
	return trackingIdentity{AppName: appName, Group: gk[0], Kind: gk[1], Namespace: nn[0], Name: nn[1]}, nil
}

func (p *Provider) resolveOwnerWithReason(ctx context.Context, workload client.Object) (gitops.Owner, gitops.GateReason, error) {
	if workload == nil {
		return gitops.Owner{}, gitops.GateOwnerUnknown, nil
	}

	group := workload.GetObjectKind().GroupVersionKind().Group
	kind := workload.GetObjectKind().GroupVersionKind().Kind
	if annotation := workload.GetAnnotations()[trackingIDAnnotation]; annotation != "" {
		identity, err := parseTrackingID(annotation)
		if err == nil && identity.Group == group && identity.Kind == kind && identity.Namespace == workload.GetNamespace() && identity.Name == workload.GetName() {
			for _, ns := range p.applicationNamespaces() {
				app, getErr := p.getApplication(ctx, types.NamespacedName{Namespace: ns, Name: identity.AppName})
				if getErr == nil {
					project, _, _ := unstructured.NestedString(app.Object, "spec", "project")
					return gitops.Owner{Provider: p.Name(), APIVersion: applicationGVK.GroupVersion().String(), Kind: applicationGVK.Kind, Namespace: ns, Name: identity.AppName, Project: project}, gitops.GateAllowed, nil
				}
			}
		}
	}

	matching, err := p.findFallbackMatches(ctx, group, kind, workload.GetNamespace(), workload.GetName())
	if err != nil {
		return gitops.Owner{}, gitops.GateProviderUnavailable, err
	}
	if len(matching) == 0 {
		return gitops.Owner{}, gitops.GateOwnerUnknown, nil
	}
	if len(matching) > 1 {
		return gitops.Owner{}, gitops.GateAmbiguousOwner, nil
	}
	app := matching[0]
	project, _, _ := unstructured.NestedString(app.Object, "spec", "project")
	return gitops.Owner{Provider: p.Name(), APIVersion: applicationGVK.GroupVersion().String(), Kind: applicationGVK.Kind, Namespace: app.GetNamespace(), Name: app.GetName(), Project: project}, gitops.GateAllowed, nil
}

func (p *Provider) applicationNamespaces() []string {
	if len(p.ApplicationNamespaces) == 0 {
		return []string{p.ControlPlaneNamespace}
	}
	return p.ApplicationNamespaces
}

func (p *Provider) getApplication(ctx context.Context, key types.NamespacedName) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(applicationGVK)
	if err := p.Client.Get(ctx, key, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (p *Provider) getAppProjectObject(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(appProjectGVK)
	if err := p.Client.Get(ctx, types.NamespacedName{Namespace: p.ControlPlaneNamespace, Name: name}, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (p *Provider) findFallbackMatches(ctx context.Context, group, kind, namespace, name string) ([]unstructured.Unstructured, error) {
	matches := make([]unstructured.Unstructured, 0)
	for _, ns := range p.applicationNamespaces() {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: applicationGVK.Group, Version: applicationGVK.Version, Kind: "ApplicationList"})
		if err := p.Client.List(ctx, list, client.InNamespace(ns)); err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			resources, found, _ := unstructured.NestedSlice(item.Object, "status", "resources")
			if !found {
				continue
			}
			if hasResource(resources, group, kind, namespace, name) {
				matches = append(matches, item)
			}
		}
	}
	return matches, nil
}

func hasResource(resources []interface{}, group, kind, namespace, name string) bool {
	for _, res := range resources {
		entry, ok := res.(map[string]interface{})
		if !ok {
			continue
		}
		if entry["group"] == group && entry["kind"] == kind && entry["namespace"] == namespace && entry["name"] == name {
			return true
		}
	}
	return false
}
