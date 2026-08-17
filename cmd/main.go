package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/apiprobe"
	"github.com/corewire/kick/internal/controller"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/executor"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	argocdprovider "github.com/corewire/kick/internal/gitops/argocd"
	fluxprovider "github.com/corewire/kick/internal/gitops/flux"
	kargoprovider "github.com/corewire/kick/internal/gitops/kargo"
	"github.com/corewire/kick/internal/integrations"
	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/notify"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	"github.com/corewire/kick/internal/rollout"
	"github.com/corewire/kick/internal/telemetry"
	"github.com/corewire/kick/internal/timeline"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kickv1alpha1.AddToScheme(scheme))
}

// options holds everything the operator is configured with on the command line.
type options struct {
	metricsAddr          string
	probeAddr            string
	timelineAddr         string
	otlpEndpoint         string
	otlpInsecure         bool
	leaderElection       bool
	requestRetention     time.Duration
	rolloutTimeout       time.Duration
	enableCSIIntegration bool
	enableArgoRollouts   bool
	providers            providerConfig
}

func parseFlags() options {
	var opts options
	var argocdApplicationNamespaces string
	flag.StringVar(&opts.metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint address.")
	flag.StringVar(&opts.probeAddr, "health-probe-bind-address", ":8081", "Health probe address.")
	flag.StringVar(&opts.timelineAddr, "timeline-bind-address", ":8090", "Timeline API/UI bind address. Empty disables the timeline server.")
	flag.BoolVar(&opts.leaderElection, "leader-elect", false, "Enable leader election.")
	flag.StringVar(&opts.otlpEndpoint, "otel-otlp-endpoint", "", "OTLP endpoint (host:port) for exporting traces to Tempo/Jaeger or another collector.")
	flag.BoolVar(&opts.otlpInsecure, "otel-otlp-insecure", true, "Use insecure OTLP transport (no TLS).")
	flag.DurationVar(&opts.requestRetention, "request-retention", 24*time.Hour, "Retention duration for terminal KickRequests before deletion.")
	flag.DurationVar(&opts.rolloutTimeout, "rollout-timeout", 15*time.Minute, "How long a restart may take before the KickRequest fails with RolloutTimeout.")
	flag.BoolVar(&opts.enableCSIIntegration, "enable-csi-integration", false, "Watch SecretProviderClassPodStatus to restart workloads when Secrets Store CSI secrets rotate. Ignored when the CRD is absent.")
	flag.BoolVar(&opts.enableArgoRollouts, "enable-argo-rollouts", false, "Treat argoproj.io Rollouts as restartable workloads. Ignored when the CRD is absent.")
	flag.BoolVar(&opts.providers.ArgoCDEnabled, "enable-argocd", true, "Gate restarts on Argo CD Application state. Ignored when the CRD is absent.")
	flag.BoolVar(&opts.providers.FluxEnabled, "enable-flux", true, "Gate restarts on Flux Kustomization and HelmRelease state. Ignored when the CRDs are absent.")
	flag.BoolVar(&opts.providers.KargoEnabled, "enable-kargo", false, "Block restarts while a Kargo Promotion is in flight. Ignored when the CRD is absent.")
	flag.StringVar(&opts.providers.ArgoCDNamespace, "argocd-namespace", "argocd", "Namespace holding Argo CD AppProjects.")
	flag.StringVar(&argocdApplicationNamespaces, "argocd-application-namespaces", "", "Comma-separated namespaces to search for Argo CD Applications. Empty means the Argo CD namespace only.")
	zapOptions := zap.Options{Development: true}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	opts.providers.ArgoCDApplicationNamespaces = splitNamespaces(argocdApplicationNamespaces)
	return opts
}

func main() {
	opts := parseFlags()
	shutdownTracing, err := telemetry.SetupOTLP(context.Background(), "kick-controller", opts.otlpEndpoint, opts.otlpInsecure)
	if err != nil {
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: opts.metricsAddr},
		HealthProbeBindAddress: opts.probeAddr,
		LeaderElection:         opts.leaderElection,
		LeaderElectionID:       "kick.corewire.io",
		Client: client.Options{
			Cache: &client.CacheOptions{
				// KICK reads these two kinds back in read-modify-write cycles:
				// observation records live in Leases, and a KickRequest is created
				// and then immediately updated. The informer cache lags behind its
				// own writes, and a stale read is not merely slow here: an
				// observation record that reads as missing re-establishes a
				// baseline, which anchors the change to the source's creation time
				// and silently swallows it. Both kinds are low-volume, so reading
				// them straight from the API server is the correct trade.
				DisableFor: []client.Object{&coordinationv1.Lease{}, &kickv1alpha1.KickRequest{}},
			},
		},
	})
	if err != nil {
		os.Exit(1)
	}
	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		<-ctx.Done()
		_ = shutdownTracing(context.Background())
		return nil
	})); err != nil {
		os.Exit(1)
	}
	if err := setupControllers(mgr, opts); err != nil {
		os.Exit(1)
	}
	if opts.timelineAddr != "" {
		if err := addTimelineServer(mgr, opts.timelineAddr); err != nil {
			os.Exit(1)
		}
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		os.Exit(1)
	}
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		os.Exit(1)
	}
}

// setupControllers wires every reconciler, including the optional workload
// kinds that may only be watched once their CRDs are known to exist.
func setupControllers(mgr ctrl.Manager, opts options) error {
	policyMatcher := &policy.DeploymentPolicyMatcher{Client: mgr.GetClient()}
	providerRegistry := newProviderRegistry(mgr, opts.providers)
	notifier := notify.NewWebhookDispatcher(mgr.GetClient(), notify.DefaultQueueSize)
	if err := mgr.Add(notifier); err != nil {
		return err
	}
	if err := (&controller.KickRequestReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		PolicyMatcher:      policyMatcher,
		GateResolver:       &controller.RegistryGateResolver{Registry: providerRegistry},
		ObservationStore:   observation.NewLeaseStore(mgr.GetClient()),
		FreshnessEvaluator: &freshness.Evaluator{Inspector: &rollout.LiveRolloutInspector{Client: mgr.GetClient()}},
		RestartExecutor:    executor.NewRestartExecutor(mgr.GetClient(), opts.rolloutTimeout),
		Notifier:           notifier,
		RequeueInterval:    30 * time.Second,
		RequestRetention:   opts.requestRetention,
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&controller.NotificationPolicyReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := (&controller.KickPolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Registry: providerRegistry,
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	// Registering an index or watch for a kind whose CRD is absent aborts the
	// manager, so the flag and the cluster must both allow it.
	var optionalWorkloadKinds []dependency.WorkloadKind
	if opts.enableArgoRollouts {
		if apiprobe.KindAvailable(mgr.GetRESTMapper(), dependency.ArgoRolloutGVK) {
			optionalWorkloadKinds = append(optionalWorkloadKinds, dependency.ArgoRolloutWorkloadKind)
		} else {
			setupLog.Info("optional integration skipped", "integration", integrations.ArgoRollouts.Title,
				"reason", integrations.ArgoRollouts.KindNotInstalledMessage(dependency.ArgoRolloutGVK))
		}
	}
	return setupObservationControllers(mgr, policyMatcher, optionalWorkloadKinds, opts.enableCSIIntegration)
}

// setupObservationControllers wires the Secret/ConfigMap observer, the reverse
// indexes, and the optional Secrets Store CSI observer.
func setupObservationControllers(
	mgr ctrl.Manager,
	policyMatcher *policy.DeploymentPolicyMatcher,
	optionalWorkloadKinds []dependency.WorkloadKind,
	enableCSIIntegration bool,
) error {
	newEnqueuer := func() *controller.KickRequestEnqueuer {
		return &controller.KickRequestEnqueuer{
			Client:        mgr.GetClient(),
			Coalescer:     kickrequest.NewCoalescer(mgr.GetClient(), kickrequest.RetentionConfig{}),
			PolicyMatcher: policyMatcher,
		}
	}

	if err := (&controller.SourceObservationReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Observer:              observation.NewObserver(observation.NewLeaseStore(mgr.GetClient()), nil),
		Enqueuer:              newEnqueuer(),
		OptionalWorkloadKinds: optionalWorkloadKinds,
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if err := dependency.RegisterWorkloadReverseIndexes(context.Background(), mgr.GetFieldIndexer(), optionalWorkloadKinds...); err != nil {
		return err
	}
	if !enableCSIIntegration {
		return nil
	}
	if !apiprobe.KindAvailable(mgr.GetRESTMapper(), controller.SecretProviderClassPodStatusGVK) {
		setupLog.Info("optional integration skipped", "integration", integrations.SecretsStoreCSI.Title,
			"reason", integrations.SecretsStoreCSI.KindNotInstalledMessage(controller.SecretProviderClassPodStatusGVK))
		return nil
	}
	return (&controller.SecretProviderClassObservationReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Observer:              observation.NewObserver(observation.NewLeaseStore(mgr.GetClient()), nil),
		Enqueuer:              newEnqueuer(),
		OptionalWorkloadKinds: optionalWorkloadKinds,
	}).SetupWithManager(mgr)
}

// splitNamespaces parses a comma-separated namespace list, ignoring blanks.
func splitNamespaces(value string) []string {
	var namespaces []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			namespaces = append(namespaces, trimmed)
		}
	}
	return namespaces
}

// newProviderRegistry builds the GitOps provider registry. A provider is only
// registered when its integration is enabled and its CRDs are served; every
// other case is recorded so a policy naming the provider can be told which
// switch to flip instead of just failing.
func newProviderRegistry(mgr ctrl.Manager, cfg providerConfig) *gitops.Registry {
	argocdProvider := &argocdprovider.Provider{
		Client:                mgr.GetClient(),
		ControlPlaneNamespace: cfg.ArgoCDNamespace,
		ApplicationNamespaces: cfg.ArgoCDApplicationNamespaces,
	}
	registry := gitops.NewRegistry()
	if register(registry, integrations.ArgoCD, cfg.ArgoCDEnabled, mgr, argocdprovider.ApplicationGVK) {
		registry.Register(argocdProvider)
	}
	if register(registry, integrations.Flux, cfg.FluxEnabled, mgr, fluxprovider.KustomizationGVK) {
		registry.Register(&fluxprovider.Provider{Client: mgr.GetClient()})
	}
	if register(registry, integrations.Kargo, cfg.KargoEnabled, mgr, kargoprovider.StageGVK) {
		registry.Register(&kargoprovider.Provider{Client: mgr.GetClient(), ArgoCD: argocdProvider})
	}
	return registry
}

// providerConfig collects the GitOps provider switches resolved from flags.
type providerConfig struct {
	ArgoCDEnabled               bool
	FluxEnabled                 bool
	KargoEnabled                bool
	ArgoCDNamespace             string
	ArgoCDApplicationNamespaces []string
}

// register reports whether an integration may be wired up, recording the reason
// in the registry when it may not.
func register(registry *gitops.Registry, integration integrations.Integration, enabled bool, mgr ctrl.Manager, gvk schema.GroupVersionKind) bool {
	switch {
	case !enabled:
		registry.MarkUnavailable(integration.Name, gitops.Unavailability{Reason: integrations.ReasonDisabled, Message: integration.DisabledMessage()})
		return false
	case !apiprobe.KindAvailable(mgr.GetRESTMapper(), gvk):
		message := integration.KindNotInstalledMessage(gvk)
		registry.MarkUnavailable(integration.Name, gitops.Unavailability{Reason: integrations.ReasonKindNotInstalled, Message: message})
		setupLog.Info("optional integration skipped", "integration", integration.Title, "reason", message)
		return false
	default:
		return true
	}
}

// addTimelineServer runs the read-only timeline API alongside the manager.
func addTimelineServer(mgr ctrl.Manager, addr string) error {
	svc := &timeline.Service{Client: mgr.GetClient(), ObservationStore: observation.NewLeaseStore(mgr.GetClient())}
	return mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		mux := http.NewServeMux()
		timeline.RegisterHandlers(mux, svc)
		srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		shutdownBase := context.Background()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(shutdownBase, 10*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}))
}
