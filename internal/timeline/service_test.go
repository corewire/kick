package timeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/observation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleDiscoveryListsAndFiltersPolicyMatchedWorkloads(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	policyAPI := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-api", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{
				Mode:             kickv1alpha1.KickPolicyDiscoveryModeAuto,
				WorkloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			},
			GitOps: kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	policyAll := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-all", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a", Labels: map[string]string{"app": "api"}}}
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "team-a", Labels: map[string]string{"app": "db"}}}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(policyAPI, policyAll, dep, sts).Build()
	svc := &Service{Client: c, ObservationStore: observation.NewMemoryStore()}
	mux := http.NewServeMux()
	RegisterHandlers(mux, svc)

	requestAll := httptest.NewRequest(http.MethodGet, "http://localhost/timeline/discovery?namespace=team-a", nil)
	responseAll := httptest.NewRecorder()
	mux.ServeHTTP(responseAll, requestAll)
	if responseAll.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", responseAll.Code)
	}

	var all DiscoveryResponse
	if err := json.Unmarshal(responseAll.Body.Bytes(), &all); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(all.Policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(all.Policies))
	}
	if len(all.Items) != 3 {
		t.Fatalf("expected 3 discovered workload entries, got %d", len(all.Items))
	}

	requestFiltered := httptest.NewRequest(http.MethodGet, "http://localhost/timeline/discovery?namespace=team-a&policy=policy-api&kind=Deployment&name=api", nil)
	responseFiltered := httptest.NewRecorder()
	mux.ServeHTTP(responseFiltered, requestFiltered)
	if responseFiltered.Code != http.StatusOK {
		t.Fatalf("expected filtered 200, got %d", responseFiltered.Code)
	}

	var filtered DiscoveryResponse
	if err := json.Unmarshal(responseFiltered.Body.Bytes(), &filtered); err != nil {
		t.Fatalf("decode filtered response: %v", err)
	}
	if len(filtered.Items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(filtered.Items))
	}
	if filtered.Items[0].Policy != "policy-api" || filtered.Items[0].Kind != "Deployment" || filtered.Items[0].Name != "api" {
		t.Fatalf("unexpected filtered item: %#v", filtered.Items[0])
	}
}

func TestHandleDAGBuildsPolicyWorkloadDependencyGraph(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	policy := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-api", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "registry.k8s.io/pause:3.10",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "api-secret"}},
						},
					},
				},
			}}},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(policy, dep).Build()
	svc := &Service{Client: c, ObservationStore: observation.NewMemoryStore()}
	mux := http.NewServeMux()
	RegisterHandlers(mux, svc)

	request := httptest.NewRequest(http.MethodGet, "http://localhost/timeline/dag?namespace=team-a", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var dag DAGResponse
	if err := json.Unmarshal(response.Body.Bytes(), &dag); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(dag.Nodes) == 0 || len(dag.Edges) == 0 {
		t.Fatalf("expected non-empty graph, got nodes=%d edges=%d", len(dag.Nodes), len(dag.Edges))
	}

	hasPolicyNode := false
	hasWorkloadNode := false
	hasSecretNode := false
	hasManageEdge := false
	hasDependsEdge := false
	for _, node := range dag.Nodes {
		switch {
		case node.ID == "policy:policy-api":
			hasPolicyNode = true
		case node.ID == "workload:Deployment:api":
			hasWorkloadNode = true
		case node.ID == "source:Secret:api-secret":
			hasSecretNode = true
		}
	}
	for _, edge := range dag.Edges {
		if edge.From == "policy:policy-api" && edge.To == "workload:Deployment:api" && edge.Type == "manages" {
			hasManageEdge = true
		}
		if edge.From == "workload:Deployment:api" && edge.To == "source:Secret:api-secret" && edge.Type == "dependsOn" {
			hasDependsEdge = true
		}
	}

	if !hasPolicyNode || !hasWorkloadNode || !hasSecretNode || !hasManageEdge || !hasDependsEdge {
		t.Fatalf("missing graph parts policy=%t workload=%t secret=%t manages=%t depends=%t", hasPolicyNode, hasWorkloadNode, hasSecretNode, hasManageEdge, hasDependsEdge)
	}
}

func TestHandleNamespacesAndResources(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	nsA := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}
	nsB := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-b"}}

	policyA := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-a", Namespace: "team-a"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	policyB := &kickv1alpha1.KickPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-b", Namespace: "team-b"},
		Spec: kickv1alpha1.KickPolicySpec{
			Discovery: kickv1alpha1.KickPolicyDiscoverySpec{Mode: kickv1alpha1.KickPolicyDiscoveryModeAuto},
			GitOps:    kickv1alpha1.KickPolicyGitOpsSpec{Provider: kickv1alpha1.KickPolicyProviderAuto},
		},
	}
	requestA := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "request-a", Namespace: "team-a"},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api-a"}},
		Status:     kickv1alpha1.KickRequestStatus{Phase: kickv1alpha1.KickRequestPhaseWaitingForOwner},
	}
	requestB := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "request-b", Namespace: "team-b"},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "DaemonSet", Name: "agent-b"}},
		Status:     kickv1alpha1.KickRequestStatus{Phase: kickv1alpha1.KickRequestPhaseExecuting},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(nsA, nsB, policyA, policyB, requestA, requestB).Build()
	svc := &Service{Client: c, ObservationStore: observation.NewMemoryStore()}
	mux := http.NewServeMux()
	RegisterHandlers(mux, svc)

	namespaceReq := httptest.NewRequest(http.MethodGet, "http://localhost/timeline/namespaces", nil)
	namespaceRes := httptest.NewRecorder()
	mux.ServeHTTP(namespaceRes, namespaceReq)
	if namespaceRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", namespaceRes.Code)
	}
	var namespaces NamespaceResponse
	if err := json.Unmarshal(namespaceRes.Body.Bytes(), &namespaces); err != nil {
		t.Fatalf("decode namespace response: %v", err)
	}
	if len(namespaces.Items) < 2 {
		t.Fatalf("expected at least 2 namespaces, got %d", len(namespaces.Items))
	}

	resourcesReq := httptest.NewRequest(http.MethodGet, "http://localhost/timeline/resources?namespace=team-a", nil)
	resourcesRes := httptest.NewRecorder()
	mux.ServeHTTP(resourcesRes, resourcesReq)
	if resourcesRes.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resourcesRes.Code)
	}
	var resources ResourcesResponse
	if err := json.Unmarshal(resourcesRes.Body.Bytes(), &resources); err != nil {
		t.Fatalf("decode resources response: %v", err)
	}
	if len(resources.Policies) != 1 || resources.Policies[0].Name != "policy-a" {
		t.Fatalf("unexpected policies payload: %#v", resources.Policies)
	}
	if len(resources.Requests) != 1 || resources.Requests[0].Name != "request-a" {
		t.Fatalf("unexpected requests payload: %#v", resources.Requests)
	}
}

func TestRegisterHandlersRootRedirectsToTimelineUI(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, &Service{})

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8090/", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/timeline/ui" {
		t.Fatalf("expected redirect to /timeline/ui, got %q", got)
	}
}

func TestBuildTimelineIncludesDependencyAndRequest(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add kube scheme: %v", err)
	}
	if err := kickv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add kick scheme: %v", err)
	}

	restartedAt := "2026-08-07T11:00:00Z"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "api"},
					Annotations: map[string]string{"kubectl.kubernetes.io/restartedAt": restartedAt},
				},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name:  "app",
					Image: "registry.k8s.io/pause:3.10",
					EnvFrom: []corev1.EnvFromSource{{
						SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-secret"}},
					}},
				}}},
			},
		},
	}
	req := &kickv1alpha1.KickRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "team-a", CreationTimestamp: metav1.NewTime(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))},
		Spec:       kickv1alpha1.KickRequestSpec{TargetRef: kickv1alpha1.ObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "api"}},
		Status:     kickv1alpha1.KickRequestStatus{Phase: kickv1alpha1.KickRequestPhaseExecuting},
	}
	event := &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "e1", Namespace: "team-a", CreationTimestamp: metav1.NewTime(time.Date(2026, 8, 7, 10, 1, 0, 0, time.UTC))}, InvolvedObject: corev1.ObjectReference{Kind: "KickRequest", Name: "api"}, Reason: "KickStarted", Message: "restart required"}

	store := observation.NewMemoryStore()
	changeAt := time.Date(2026, 8, 7, 9, 59, 0, 0, time.UTC)
	if err := store.Upsert(context.Background(), observation.Record{Identity: observation.SourceIdentity{APIVersion: "v1", Kind: observation.SourceKindSecret, Namespace: "team-a", Name: "app-secret"}, LastRelevantChangeTime: changeAt}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep, req, event).Build()
	svc := &Service{Client: c, ObservationStore: store}

	items, err := svc.buildTimeline(context.Background(), "team-a", "Deployment", "api")
	if err != nil {
		t.Fatalf("build timeline: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("expected timeline entries, got %d", len(items))
	}

	foundDependency := false
	foundRestart := false
	for _, item := range items {
		if item.Type == "DependencyRelevantChange" && item.Object == "Secret/team-a/app-secret" {
			foundDependency = true
		}
		if item.Type == "WorkloadRestarted" {
			foundRestart = true
		}
	}
	if !foundDependency {
		t.Fatalf("missing dependency-change entry")
	}
	if !foundRestart {
		t.Fatalf("missing workload-restarted entry")
	}
}
