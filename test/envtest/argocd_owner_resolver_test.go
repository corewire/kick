package envtest

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/corewire/kick/internal/gitops"
	"github.com/corewire/kick/internal/gitops/argocd"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestArgoCDOwnerResolverEnvtest(t *testing.T) {
	t.Parallel()
	env := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")}}
	defer func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("stop envtest: %v", err)
		}
	}()
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}

	ctx := context.Background()
	apiExtClient, err := apiextensionsclient.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("new apiextensions client: %v", err)
	}
	for _, crd := range []*apiextensionsv1.CustomResourceDefinition{minimalArgoprojCRD("applications", "Application"), minimalArgoprojCRD("appprojects", "AppProject")} {
		if _, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{}); err != nil {
			t.Fatalf("create crd %s: %v", crd.Name, err)
		}
		if err := wait.PollUntilContextTimeout(ctx, 200*time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
			got, getErr := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
			if getErr != nil {
				return false, getErr
			}
			for _, cond := range got.Status.Conditions {
				if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
					return true, nil
				}
			}
			return false, nil
		}); err != nil {
			t.Fatalf("wait crd established %s: %v", crd.Name, err)
		}
	}

	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	for _, ns := range []string{"argocd", "team-a", "team-b", "payments"} {
		if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil {
			t.Fatalf("create namespace %s: %v", ns, err)
		}
	}

	app := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      "payments-app",
			"namespace": "argocd",
		},
		"spec":   map[string]interface{}{"project": "production"},
		"status": map[string]interface{}{"resources": []interface{}{map[string]interface{}{"group": "apps", "kind": "Deployment", "namespace": "payments", "name": "payments-api"}}},
	}}
	app.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"})
	if err := c.Create(ctx, app); err != nil {
		t.Fatalf("create application: %v", err)
	}

	project := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "AppProject",
		"metadata":   map[string]interface{}{"name": "production", "namespace": "argocd"},
	}}
	project.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "AppProject"})
	if err := c.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	provider := &argocd.Provider{Client: c, ApplicationNamespaces: []string{"argocd", "team-a", "team-b"}, ControlPlaneNamespace: "argocd"}
	workload := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      "payments-api",
		Namespace: "payments",
		Annotations: map[string]string{
			"argocd.argoproj.io/tracking-id": "payments-app:apps/Deployment:payments/payments-api",
		},
	}}
	workload.SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})

	owner, err := provider.ResolveOwner(ctx, workload)
	if err != nil {
		t.Fatalf("resolve owner: %v", err)
	}
	if owner.Namespace != "argocd" || owner.Name != "payments-app" {
		t.Fatalf("owner mismatch: %#v", owner)
	}

	workload.Annotations["argocd.argoproj.io/tracking-id"] = "payments-app:apps/Deployment:other/payments-api"
	owner, err = provider.ResolveOwner(ctx, workload)
	if err != nil {
		t.Fatalf("fallback resolve owner: %v", err)
	}
	if owner.Namespace != "argocd" || owner.Name != "payments-app" {
		t.Fatalf("fallback owner mismatch: %#v", owner)
	}

	annotatedApp := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      "annotated-app",
			"namespace": "team-a",
		},
		"spec": map[string]interface{}{"project": "production"},
	}}
	annotatedApp.SetGroupVersionKind(schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"})
	if err := c.Create(ctx, annotatedApp); err != nil {
		t.Fatalf("create annotated application: %v", err)
	}

	workload.Annotations["argocd.argoproj.io/tracking-id"] = "annotated-app:apps/Deployment:payments/payments-api"
	if _, err := provider.ResolveOwner(ctx, workload); err == nil {
		t.Fatalf("expected annotation owner outside control-plane namespace to block")
	} else {
		var resolutionErr argocd.ResolutionError
		if !errors.As(err, &resolutionErr) || resolutionErr.Reason != gitops.GateOwnerUnknown {
			t.Fatalf("unexpected resolution error: %v", err)
		}
	}

	if err := unstructured.SetNestedSlice(app.Object, []interface{}{}, "status", "resources"); err != nil {
		t.Fatalf("set empty resources: %v", err)
	}
	if err := c.Update(ctx, app); err != nil {
		t.Fatalf("update application resources: %v", err)
	}
	if _, err := provider.ResolveOwner(ctx, workload); err == nil {
		t.Fatalf("expected typed owner resolution block")
	} else {
		var resolutionErr argocd.ResolutionError
		if !errors.As(err, &resolutionErr) || resolutionErr.Reason != gitops.GateOwnerUnknown {
			t.Fatalf("unexpected resolution error: %v", err)
		}
	}
}

func minimalArgoprojCRD(plural, kind string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: plural + ".argoproj.io"},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "argoproj.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: plural, Singular: strings.ToLower(kind), Kind: kind},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema:  &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object", XPreserveUnknownFields: pointerBool(true)}},
			}},
		},
	}
}

func pointerBool(v bool) *bool { return &v }
