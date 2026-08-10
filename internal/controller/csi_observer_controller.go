package controller

import (
	"context"
	"sort"
	"time"

	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/observation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	secretProviderClassPodStatusKind     = "SecretProviderClassPodStatus"
	secretProviderClassPodStatusListKind = "SecretProviderClassPodStatusList"

	// csiInconsistentRequeue is how long to wait before re-checking a
	// SecretProviderClass whose pods still disagree about the mounted versions.
	csiInconsistentRequeue = 30 * time.Second
)

// SecretProviderClassPodStatusGVK is the GroupVersionKind observed for external
// secret freshness.
var SecretProviderClassPodStatusGVK = schema.GroupVersionKind{
	Group:   "secrets-store.csi.x-k8s.io",
	Version: "v1",
	Kind:    secretProviderClassPodStatusKind,
}

// SecretProviderClassObservationReconciler derives freshness for external
// secrets mounted through the Secrets Store CSI driver.
//
// The driver never writes the secret material into the API server, so the only
// available signal is SecretProviderClassPodStatus, which reports the provider
// object versions each pod currently has mounted. Requests are keyed by the
// SecretProviderClass, not by pod, so a workload is looked up directly through
// the SecretProviderClass reverse index instead of through its pods.
type SecretProviderClassObservationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Observer *observation.Observer
	Enqueuer ConsumerRequestEnqueuer
	// OptionalWorkloadKinds are CRD-backed workload kinds (for example Argo
	// Rollouts) that are only looked up when their CRD is installed.
	OptionalWorkloadKinds []dependency.WorkloadKind
}

// Reconcile receives <namespace>/<secretProviderClassName>, not the name of a
// SecretProviderClassPodStatus.
func (r *SecretProviderClassObservationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	mounts, err := r.collectMounts(ctx, req.Namespace, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(mounts) == 0 {
		// Every pod unmounted the class. Deletions are not a content change, so
		// nothing is observed and no restart is issued.
		return ctrl.Result{}, nil
	}

	observedAt := time.Now().UTC()
	result, err := r.Observer.ObserveSecretProviderClass(ctx, req.Namespace, req.Name, mounts, observedAt)
	if err != nil {
		return ctrl.Result{}, err
	}
	if result.Kind == observation.NoChange {
		// Either genuinely unchanged, or mid-rotation with pods still disagreeing.
		// Re-check so a rotation that settles is not missed.
		return ctrl.Result{RequeueAfter: csiInconsistentRequeue}, nil
	}
	if result.Kind != observation.RelevantChange {
		return ctrl.Result{}, nil
	}

	consumers, err := dependency.LookupConsumingWorkloads(ctx, r.Client, dependency.DependencyRef{
		APIVersion: dependency.SecretsStoreAPIVersion,
		Kind:       dependency.SecretProviderClass,
		Namespace:  req.Namespace,
		Name:       req.Name,
	}, r.OptionalWorkloadKinds...)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(consumers) == 0 {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.Enqueuer.EnqueueForConsumers(ctx, result.Identity, r.classLabels(ctx, req.Namespace, req.Name), consumers, observedAt)
}

// classLabels returns the labels of the SecretProviderClass so dependency
// selectors keep working. A missing class yields no labels rather than an error.
func (r *SecretProviderClassObservationReconciler) classLabels(ctx context.Context, namespace, name string) map[string]string {
	var class unstructured.Unstructured
	class.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   SecretProviderClassPodStatusGVK.Group,
		Version: SecretProviderClassPodStatusGVK.Version,
		Kind:    "SecretProviderClass",
	})
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &class); err != nil {
		return nil
	}
	return class.GetLabels()
}

// collectMounts returns the mounted object versions reported by every pod that
// currently mounts the given SecretProviderClass in the namespace.
func (r *SecretProviderClassObservationReconciler) collectMounts(ctx context.Context, namespace, className string) ([]observation.PodMount, error) {
	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(SecretProviderClassPodStatusGVK.GroupVersion().WithKind(secretProviderClassPodStatusListKind))
	if err := r.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	mounts := make([]observation.PodMount, 0, len(list.Items))
	for i := range list.Items {
		mount, ok := podMountFrom(&list.Items[i], className)
		if !ok {
			continue
		}
		mounts = append(mounts, mount)
	}
	sort.Slice(mounts, func(i, j int) bool { return mounts[i].PodName < mounts[j].PodName })
	return mounts, nil
}

// podMountFrom converts one SecretProviderClassPodStatus into a PodMount.
// Statuses for a different class, or that are not mounted, are skipped.
func podMountFrom(obj *unstructured.Unstructured, className string) (observation.PodMount, bool) {
	name, _, _ := unstructured.NestedString(obj.Object, "status", "secretProviderClassName")
	if name != className {
		return observation.PodMount{}, false
	}
	if mounted, found, _ := unstructured.NestedBool(obj.Object, "status", "mounted"); found && !mounted {
		return observation.PodMount{}, false
	}
	podName, _, _ := unstructured.NestedString(obj.Object, "status", "podName")
	rawObjects, _, _ := unstructured.NestedSlice(obj.Object, "status", "objects")

	objects := make([]observation.MountedObject, 0, len(rawObjects))
	for _, raw := range rawObjects {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := entry["id"].(string)
		version, _ := entry["version"].(string)
		objects = append(objects, observation.MountedObject{ID: id, Version: version})
	}
	return observation.PodMount{PodName: podName, Objects: objects}, true
}

func (r *SecretProviderClassObservationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	source := &unstructured.Unstructured{}
	source.SetGroupVersionKind(SecretProviderClassPodStatusGVK)

	return ctrl.NewControllerManagedBy(mgr).
		Named("secretproviderclass-observation").
		Watches(source, handler.EnqueueRequestsFromMapFunc(mapPodStatusToSecretProviderClass)).
		Complete(r)
}

// mapPodStatusToSecretProviderClass collapses per-pod statuses onto the
// SecretProviderClass they report on, so N pods produce one reconcile key.
func mapPodStatusToSecretProviderClass(_ context.Context, obj client.Object) []reconcile.Request {
	raw, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	className, _, _ := unstructured.NestedString(raw.Object, "status", "secretProviderClassName")
	if className == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: raw.GetNamespace(), Name: className}}}
}
