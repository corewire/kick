package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Entry struct {
	At      time.Time `json:"at"`
	Type    string    `json:"type"`
	Object  string    `json:"object"`
	Reason  string    `json:"reason,omitempty"`
	Message string    `json:"message"`
}

type Response struct {
	Namespace string  `json:"namespace"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Items     []Entry `json:"items"`
}

type DiscoveredWorkload struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Policy    string `json:"policy"`
}

type DiscoveryResponse struct {
	Namespace string               `json:"namespace"`
	Policies  []string             `json:"policies"`
	Items     []DiscoveredWorkload `json:"items"`
}

type NamespaceResponse struct {
	Items []string `json:"items"`
}

type KickPolicySummary struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type KickRequestSummary struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Phase      string `json:"phase"`
	GateReason string `json:"gateReason,omitempty"`
	TargetKind string `json:"targetKind"`
	TargetName string `json:"targetName"`
}

type ResourcesResponse struct {
	Namespace string               `json:"namespace,omitempty"`
	Policies  []KickPolicySummary  `json:"policies"`
	Requests  []KickRequestSummary `json:"requests"`
}

type DAGNode struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type DAGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type DAGResponse struct {
	Namespace string    `json:"namespace"`
	Nodes     []DAGNode `json:"nodes"`
	Edges     []DAGEdge `json:"edges"`
}

type OverviewEvent struct {
	At        time.Time `json:"at"`
	Namespace string    `json:"namespace"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Reason    string    `json:"reason,omitempty"`
	Message   string    `json:"message"`
}

type OverviewWorkload struct {
	Namespace  string          `json:"namespace"`
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	Policy     string          `json:"policy"`
	Phase      string          `json:"phase,omitempty"`
	GateReason string          `json:"gateReason,omitempty"`
	Events     []OverviewEvent `json:"events"`
}

type OverviewResponse struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	Workloads   []OverviewWorkload `json:"workloads"`
	Events      []OverviewEvent    `json:"events"`
}

type Service struct {
	Client           client.Client
	ObservationStore observation.Store
}

func RegisterHandlers(mux *http.ServeMux, svc *Service) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/timeline/ui", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/timeline", svc.handleTimeline)
	mux.HandleFunc("/timeline/namespaces", svc.handleNamespaces)
	mux.HandleFunc("/timeline/resources", svc.handleResources)
	mux.HandleFunc("/timeline/discovery", svc.handleDiscovery)
	mux.HandleFunc("/timeline/dag", svc.handleDAG)
	mux.HandleFunc("/timeline/overview", svc.handleOverview)
	mux.HandleFunc("/timeline/ui", serveUI)
}

func (s *Service) handleNamespaces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var namespaces corev1.NamespaceList
	if err := s.Client.List(ctx, &namespaces); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	items := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		items = append(items, ns.Name)
	}
	sort.Strings(items)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(NamespaceResponse{Items: items})
}

func (s *Service) handleResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namespace := strings.TrimSpace(r.URL.Query().Get("namespace"))

	listOptions := make([]client.ListOption, 0, 1)
	if namespace != "" {
		listOptions = append(listOptions, client.InNamespace(namespace))
	}

	var policyList kickv1alpha1.KickPolicyList
	if err := s.Client.List(ctx, &policyList, listOptions...); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	policies := make([]KickPolicySummary, 0, len(policyList.Items))
	for _, pol := range policyList.Items {
		policies = append(policies, KickPolicySummary{Namespace: pol.Namespace, Name: pol.Name})
	}
	sort.Slice(policies, func(i, j int) bool {
		if policies[i].Namespace != policies[j].Namespace {
			return policies[i].Namespace < policies[j].Namespace
		}
		return policies[i].Name < policies[j].Name
	})

	var requestList kickv1alpha1.KickRequestList
	if err := s.Client.List(ctx, &requestList, listOptions...); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	requests := make([]KickRequestSummary, 0, len(requestList.Items))
	for _, req := range requestList.Items {
		requests = append(requests, KickRequestSummary{
			Namespace:  req.Namespace,
			Name:       req.Name,
			Phase:      string(req.Status.Phase),
			GateReason: req.Status.Gate.Reason,
			TargetKind: req.Spec.TargetRef.Kind,
			TargetName: req.Spec.TargetRef.Name,
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].Namespace != requests[j].Namespace {
			return requests[i].Namespace < requests[j].Namespace
		}
		if requests[i].TargetKind != requests[j].TargetKind {
			return requests[i].TargetKind < requests[j].TargetKind
		}
		if requests[i].TargetName != requests[j].TargetName {
			return requests[i].TargetName < requests[j].TargetName
		}
		return requests[i].Name < requests[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ResourcesResponse{Namespace: namespace, Policies: policies, Requests: requests})
}

func (s *Service) handleTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "Deployment"
	}
	if namespace == "" || name == "" {
		http.Error(w, "namespace and name are required", http.StatusBadRequest)
		return
	}

	items, err := s.buildTimeline(ctx, namespace, kind, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{Namespace: namespace, Name: name, Kind: kind, Items: items})
}

func (s *Service) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace is required", http.StatusBadRequest)
		return
	}

	items, policies, err := s.discoverManagedWorkloads(r.Context(), namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policyFilter := r.URL.Query().Get("policy")
	kindFilter := r.URL.Query().Get("kind")
	nameFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("name")))

	filtered := make([]DiscoveredWorkload, 0, len(items))
	for _, item := range items {
		if policyFilter != "" && item.Policy != policyFilter {
			continue
		}
		if kindFilter != "" && kindFilter != "All" && item.Kind != kindFilter {
			continue
		}
		if nameFilter != "" && !strings.Contains(strings.ToLower(item.Name), nameFilter) {
			continue
		}
		filtered = append(filtered, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DiscoveryResponse{Namespace: namespace, Policies: policies, Items: filtered})
}

func (s *Service) handleDAG(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace is required", http.StatusBadRequest)
		return
	}

	dag, err := s.buildDAG(r.Context(), namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dag)
}

// handleOverview aggregates managed workloads and their timeline events across
// every namespace into a single, time-sorted view so the UI can answer "what
// happened when" without drilling into one workload at a time.
func (s *Service) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var namespaces corev1.NamespaceList
	if err := s.Client.List(ctx, &namespaces); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workloads := make([]OverviewWorkload, 0)
	allEvents := make([]OverviewEvent, 0)
	seen := map[string]struct{}{}

	for _, ns := range namespaces.Items {
		for _, workload := range s.overviewForNamespace(ctx, ns.Name) {
			identity := workload.Namespace + "/" + workload.Kind + "/" + workload.Name
			if _, ok := seen[identity]; ok {
				continue
			}
			seen[identity] = struct{}{}
			workloads = append(workloads, workload)
			allEvents = append(allEvents, workload.Events...)
		}
	}

	sort.Slice(workloads, func(i, j int) bool {
		if workloads[i].Namespace != workloads[j].Namespace {
			return workloads[i].Namespace < workloads[j].Namespace
		}
		if workloads[i].Kind != workloads[j].Kind {
			return workloads[i].Kind < workloads[j].Kind
		}
		return workloads[i].Name < workloads[j].Name
	})

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].At.After(allEvents[j].At)
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(OverviewResponse{GeneratedAt: time.Now().UTC(), Workloads: workloads, Events: allEvents})
}

// overviewForNamespace returns the managed workloads in a namespace, each with
// its timeline events and current KickRequest phase. Namespaces without any
// managed workloads yield an empty slice.
func (s *Service) overviewForNamespace(ctx context.Context, namespace string) []OverviewWorkload {
	discovered, _, err := s.discoverManagedWorkloads(ctx, namespace)
	if err != nil || len(discovered) == 0 {
		return nil
	}

	phaseByTarget := s.requestPhaseMap(ctx, namespace)

	workloads := make([]OverviewWorkload, 0, len(discovered))
	for _, workload := range discovered {
		items, err := s.buildTimeline(ctx, workload.Namespace, workload.Kind, workload.Name)
		if err != nil {
			continue
		}
		summary := phaseByTarget[workload.Kind+"/"+workload.Name]
		workloads = append(workloads, OverviewWorkload{
			Namespace:  workload.Namespace,
			Kind:       workload.Kind,
			Name:       workload.Name,
			Policy:     workload.Policy,
			Phase:      summary.Phase,
			GateReason: summary.GateReason,
			Events:     overviewEvents(workload, items),
		})
	}
	return workloads
}

// requestPhaseMap indexes the current KickRequest phase by "Kind/Name" target.
func (s *Service) requestPhaseMap(ctx context.Context, namespace string) map[string]KickRequestSummary {
	phaseByTarget := map[string]KickRequestSummary{}
	var requestList kickv1alpha1.KickRequestList
	if err := s.Client.List(ctx, &requestList, client.InNamespace(namespace)); err != nil {
		return phaseByTarget
	}
	for _, req := range requestList.Items {
		key := req.Spec.TargetRef.Kind + "/" + req.Spec.TargetRef.Name
		phaseByTarget[key] = KickRequestSummary{Phase: string(req.Status.Phase), GateReason: req.Status.Gate.Reason}
	}
	return phaseByTarget
}

func overviewEvents(workload DiscoveredWorkload, items []Entry) []OverviewEvent {
	events := make([]OverviewEvent, 0, len(items))
	for _, item := range items {
		events = append(events, OverviewEvent{
			At:        item.At.UTC(),
			Namespace: workload.Namespace,
			Kind:      workload.Kind,
			Name:      workload.Name,
			Type:      item.Type,
			Reason:    item.Reason,
			Message:   item.Message,
		})
	}
	return events
}

func (s *Service) buildTimeline(ctx context.Context, namespace, kind, name string) ([]Entry, error) {
	workload, err := loadWorkload(ctx, s.Client, namespace, kind, name)
	if err != nil {
		return nil, err
	}

	items := make([]Entry, 0)
	for _, dep := range dependency.ExtractDependenciesForObject(workload) {
		record, found, err := s.ObservationStore.Get(ctx, observation.SourceIdentity{APIVersion: dep.APIVersion, Kind: sourceKind(dep.Kind), Namespace: dep.Namespace, Name: dep.Name})
		if err != nil {
			return nil, err
		}
		if !found || record.LastRelevantChangeTime.IsZero() {
			continue
		}
		items = append(items, Entry{At: record.LastRelevantChangeTime.UTC(), Type: "DependencyRelevantChange", Object: fmt.Sprintf("%s/%s/%s", dep.Kind, dep.Namespace, dep.Name), Message: "relevant dependency content change observed"})
	}

	restartedAt := workloadRestartedAt(workload)
	if restartedAt != nil {
		items = append(items, Entry{At: restartedAt.UTC(), Type: "WorkloadRestarted", Object: fmt.Sprintf("%s/%s/%s", kind, namespace, name), Message: "pod template restartedAt annotation updated"})
	}

	requestNames, requestItems, err := relatedKickRequests(ctx, s.Client, namespace, kind, name)
	if err != nil {
		return nil, err
	}
	items = append(items, requestItems...)

	eventItems, err := relatedEvents(ctx, s.Client, namespace, kind, name, requestNames)
	if err != nil {
		return nil, err
	}
	items = append(items, eventItems...)

	sort.Slice(items, func(i, j int) bool {
		if items[i].At.Equal(items[j].At) {
			if items[i].Type != items[j].Type {
				return items[i].Type < items[j].Type
			}
			return items[i].Object < items[j].Object
		}
		return items[i].At.Before(items[j].At)
	})

	return items, nil
}

func loadWorkload(ctx context.Context, c client.Client, namespace, kind, name string) (client.Object, error) {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	switch kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := c.Get(ctx, key, &deployment); err != nil {
			return nil, err
		}
		return &deployment, nil
	case "StatefulSet":
		var statefulSet appsv1.StatefulSet
		if err := c.Get(ctx, key, &statefulSet); err != nil {
			return nil, err
		}
		return &statefulSet, nil
	case "DaemonSet":
		var daemonSet appsv1.DaemonSet
		if err := c.Get(ctx, key, &daemonSet); err != nil {
			return nil, err
		}
		return &daemonSet, nil
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func relatedKickRequests(ctx context.Context, c client.Client, namespace, kind, name string) (map[string]struct{}, []Entry, error) {
	requestNames := map[string]struct{}{}
	items := make([]Entry, 0)
	var list kickv1alpha1.KickRequestList
	if err := c.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, nil, err
	}
	for _, req := range list.Items {
		if req.Spec.TargetRef.APIVersion != "apps/v1" || req.Spec.TargetRef.Kind != kind || req.Spec.TargetRef.Name != name {
			continue
		}
		requestNames[req.Name] = struct{}{}
		items = append(items, Entry{At: req.CreationTimestamp.UTC(), Type: "KickRequestCreated", Object: fmt.Sprintf("KickRequest/%s/%s", req.Namespace, req.Name), Message: "kick request created"})
		if req.Status.Phase != "" {
			items = append(items, Entry{At: phaseTransitionTime(req.Status, req.CreationTimestamp), Type: "KickRequestPhase", Object: fmt.Sprintf("KickRequest/%s/%s", req.Namespace, req.Name), Reason: req.Status.Gate.Reason, Message: string(req.Status.Phase)})
		}
	}
	return requestNames, items, nil
}

func relatedEvents(ctx context.Context, c client.Client, namespace, kind, name string, requestNames map[string]struct{}) ([]Entry, error) {
	var events corev1.EventList
	if err := c.List(ctx, &events, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	items := make([]Entry, 0)
	for _, event := range events.Items {
		involved := event.InvolvedObject
		if involved.Kind == kind && involved.Name == name {
			items = append(items, Entry{At: eventTime(event), Type: "KubernetesEvent", Object: fmt.Sprintf("%s/%s/%s", involved.Kind, namespace, involved.Name), Reason: event.Reason, Message: event.Message})
			continue
		}
		if involved.Kind == "KickRequest" {
			if _, ok := requestNames[involved.Name]; ok {
				items = append(items, Entry{At: eventTime(event), Type: "KickRequestEvent", Object: fmt.Sprintf("KickRequest/%s/%s", namespace, involved.Name), Reason: event.Reason, Message: event.Message})
			}
		}
	}
	return items, nil
}

func eventTime(event corev1.Event) time.Time {
	if !event.EventTime.IsZero() {
		return event.EventTime.UTC()
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.UTC()
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.UTC()
	}
	return event.CreationTimestamp.UTC()
}

func phaseTransitionTime(status kickv1alpha1.KickRequestStatus, created metav1.Time) time.Time {
	for _, cond := range status.Conditions {
		if cond.Type == "Progressing" && !cond.LastTransitionTime.IsZero() {
			return cond.LastTransitionTime.UTC()
		}
	}
	return created.UTC()
}

func sourceKind(kind dependency.Kind) observation.SourceKind {
	if kind == dependency.ConfigMap {
		return observation.SourceKindConfigMap
	}
	return observation.SourceKindSecret
}

func workloadRestartedAt(obj client.Object) *time.Time {
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		return parseRestartedAt(workload.Spec.Template.Annotations)
	case *appsv1.StatefulSet:
		return parseRestartedAt(workload.Spec.Template.Annotations)
	case *appsv1.DaemonSet:
		return parseRestartedAt(workload.Spec.Template.Annotations)
	default:
		return nil
	}
}

func parseRestartedAt(annotations map[string]string) *time.Time {
	if annotations == nil {
		return nil
	}
	raw := annotations["kubectl.kubernetes.io/restartedAt"]
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	copy := parsed.UTC()
	return &copy
}

// workloadRef is a kind-tagged handle to a namespaced workload object.
type workloadRef struct {
	kind string
	name string
	obj  client.Object
}

// listWorkloads returns every Deployment, StatefulSet, and DaemonSet in a namespace.
func (s *Service) listWorkloads(ctx context.Context, namespace string) ([]workloadRef, error) {
	var deployments appsv1.DeploymentList
	if err := s.Client.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	var statefulSets appsv1.StatefulSetList
	if err := s.Client.List(ctx, &statefulSets, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	var daemonSets appsv1.DaemonSetList
	if err := s.Client.List(ctx, &daemonSets, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	refs := make([]workloadRef, 0, len(deployments.Items)+len(statefulSets.Items)+len(daemonSets.Items))
	for i := range deployments.Items {
		refs = append(refs, workloadRef{kind: "Deployment", name: deployments.Items[i].Name, obj: deployments.Items[i].DeepCopy()})
	}
	for i := range statefulSets.Items {
		refs = append(refs, workloadRef{kind: "StatefulSet", name: statefulSets.Items[i].Name, obj: statefulSets.Items[i].DeepCopy()})
	}
	for i := range daemonSets.Items {
		refs = append(refs, workloadRef{kind: "DaemonSet", name: daemonSets.Items[i].Name, obj: daemonSets.Items[i].DeepCopy()})
	}
	return refs, nil
}

func (s *Service) discoverManagedWorkloads(ctx context.Context, namespace string) ([]DiscoveredWorkload, []string, error) {
	var policyList kickv1alpha1.KickPolicyList
	if err := s.Client.List(ctx, &policyList, client.InNamespace(namespace)); err != nil {
		return nil, nil, err
	}

	policies := make([]string, 0, len(policyList.Items))
	for _, policy := range policyList.Items {
		policies = append(policies, policy.Name)
	}
	sort.Strings(policies)

	if len(policyList.Items) == 0 {
		return []DiscoveredWorkload{}, policies, nil
	}

	workloads, err := s.listWorkloads(ctx, namespace)
	if err != nil {
		return nil, nil, err
	}

	items := make([]DiscoveredWorkload, 0)
	for _, policy := range policyList.Items {
		selector, err := selectorForWorkloadSelector(policy.Spec.Discovery.WorkloadSelector)
		if err != nil {
			continue
		}
		for _, workload := range workloads {
			if selector.Matches(labels.Set(workload.obj.GetLabels())) {
				items = append(items, DiscoveredWorkload{Namespace: namespace, Kind: workload.kind, Name: workload.name, Policy: policy.Name})
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Policy != items[j].Policy {
			return items[i].Policy < items[j].Policy
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].Name < items[j].Name
	})

	return items, policies, nil
}

func selectorForWorkloadSelector(selector *metav1.LabelSelector) (labels.Selector, error) {
	if selector == nil {
		return labels.Everything(), nil
	}
	return metav1.LabelSelectorAsSelector(selector)
}

// dagBuilder accumulates unique nodes and edges for the namespace DAG.
type dagBuilder struct {
	nodes    []DAGNode
	edges    []DAGEdge
	nodeSeen map[string]struct{}
	edgeSeen map[string]struct{}
}

func newDAGBuilder() *dagBuilder {
	return &dagBuilder{nodeSeen: map[string]struct{}{}, edgeSeen: map[string]struct{}{}}
}

func (b *dagBuilder) addNode(id, kind, label string) {
	if _, found := b.nodeSeen[id]; found {
		return
	}
	b.nodeSeen[id] = struct{}{}
	b.nodes = append(b.nodes, DAGNode{ID: id, Kind: kind, Label: label})
}

func (b *dagBuilder) addEdge(from, to, edgeType string) {
	key := from + "->" + to + ":" + edgeType
	if _, found := b.edgeSeen[key]; found {
		return
	}
	b.edgeSeen[key] = struct{}{}
	b.edges = append(b.edges, DAGEdge{From: from, To: to, Type: edgeType})
}

// addPolicy links a KickPolicy to the workloads it selects and their dependency sources.
func (b *dagBuilder) addPolicy(policy kickv1alpha1.KickPolicy, workloads []workloadRef) {
	selector, err := selectorForWorkloadSelector(policy.Spec.Discovery.WorkloadSelector)
	if err != nil {
		return
	}

	policyID := "policy:" + policy.Name
	b.addNode(policyID, "KickPolicy", policy.Name)

	for _, workload := range workloads {
		if !selector.Matches(labels.Set(workload.obj.GetLabels())) {
			continue
		}
		workloadID := "workload:" + workload.kind + ":" + workload.name
		b.addNode(workloadID, workload.kind, workload.kind+"/"+workload.name)
		b.addEdge(policyID, workloadID, "manages")

		for _, dep := range dependency.ExtractDependenciesForObject(workload.obj) {
			sourceID := "source:" + string(dep.Kind) + ":" + dep.Name
			b.addNode(sourceID, string(dep.Kind), string(dep.Kind)+"/"+dep.Name)
			b.addEdge(workloadID, sourceID, "dependsOn")
		}
	}
}

func (s *Service) buildDAG(ctx context.Context, namespace string) (DAGResponse, error) {
	var policyList kickv1alpha1.KickPolicyList
	if err := s.Client.List(ctx, &policyList, client.InNamespace(namespace)); err != nil {
		return DAGResponse{}, err
	}

	workloads, err := s.listWorkloads(ctx, namespace)
	if err != nil {
		return DAGResponse{}, err
	}

	builder := newDAGBuilder()
	for _, policy := range policyList.Items {
		builder.addPolicy(policy, workloads)
	}

	nodes := builder.nodes
	edges := builder.edges
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Label < nodes[j].Label
	})
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Type < edges[j].Type
	})

	return DAGResponse{Namespace: namespace, Nodes: nodes, Edges: edges}, nil
}
