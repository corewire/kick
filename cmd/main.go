package main

import (
	"context"
	"flag"
	"net/http"
	"os"
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
	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/notify"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	"github.com/corewire/kick/internal/rollout"
	"github.com/corewire/kick/internal/telemetry"
	"github.com/corewire/kick/internal/timeline"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kickv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr, probeAddr string
	var leaderElection bool
	var requestRetention time.Duration
	var timelineAddr string
	var otlpEndpoint string
	var otlpInsecure bool
	var enableCSIIntegration bool
	var enableArgoRollouts bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint address.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe address.")
	flag.StringVar(&timelineAddr, "timeline-bind-address", ":8090", "Timeline API/UI bind address. Empty disables the timeline server.")
	flag.BoolVar(&leaderElection, "leader-elect", false, "Enable leader election.")
	flag.StringVar(&otlpEndpoint, "otel-otlp-endpoint", "", "OTLP endpoint (host:port) for exporting traces to Tempo/Jaeger or another collector.")
	flag.BoolVar(&otlpInsecure, "otel-otlp-insecure", true, "Use insecure OTLP transport (no TLS).")
	flag.DurationVar(&requestRetention, "request-retention", 24*time.Hour, "Retention duration for terminal KickRequests before deletion.")
	flag.BoolVar(&enableCSIIntegration, "enable-csi-integration", false, "Watch SecretProviderClassPodStatus to restart workloads when Secrets Store CSI secrets rotate. Ignored when the CRD is absent.")
	flag.BoolVar(&enableArgoRollouts, "enable-argo-rollouts", false, "Treat argoproj.io Rollouts as restartable workloads. Ignored when the CRD is absent.")
	zapOptions := zap.Options{Development: true}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	shutdownTracing, err := telemetry.SetupOTLP(context.Background(), "kick-controller", otlpEndpoint, otlpInsecure)
	if err != nil {
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         leaderElection,
		LeaderElectionID:       "kick.corewire.io",
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
	policyMatcher := &policy.DeploymentPolicyMatcher{Client: mgr.GetClient()}
	providerRegistry := newProviderRegistry(mgr)
	notifier := notify.NewWebhookDispatcher(mgr.GetClient(), notify.DefaultQueueSize)
	if err := mgr.Add(notifier); err != nil {
		os.Exit(1)
	}
	if err := (&controller.KickRequestReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		PolicyMatcher:      policyMatcher,
		GateResolver:       &controller.RegistryGateResolver{Registry: providerRegistry},
		ObservationStore:   observation.NewLeaseStore(mgr.GetClient()),
		FreshnessEvaluator: &freshness.Evaluator{Inspector: &rollout.LiveRolloutInspector{Client: mgr.GetClient()}},
		RestartExecutor:    executor.NewRestartExecutor(mgr.GetClient(), 10*time.Minute),
		Notifier:           notifier,
		RequeueInterval:    30 * time.Second,
		RequestRetention:   requestRetention,
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}
	// Optional integrations: registering an index or watch for a kind whose CRD
	// is absent aborts the manager, so the flag and the cluster must both allow it.
	var optionalWorkloadKinds []dependency.WorkloadKind
	if enableArgoRollouts && apiprobe.KindAvailable(mgr.GetRESTMapper(), dependency.ArgoRolloutGVK) {
		optionalWorkloadKinds = append(optionalWorkloadKinds, dependency.ArgoRolloutWorkloadKind)
	}

	if err := setupObservationControllers(mgr, policyMatcher, optionalWorkloadKinds, enableCSIIntegration); err != nil {
		os.Exit(1)
	}
	if timelineAddr != "" {
		if err := addTimelineServer(mgr, timelineAddr); err != nil {
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
	if !enableCSIIntegration || !apiprobe.KindAvailable(mgr.GetRESTMapper(), controller.SecretProviderClassPodStatusGVK) {
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

// newProviderRegistry builds the GitOps provider registry. Kargo is only
// registered when its CRDs exist, because a policy naming an unavailable
// provider must fail fast rather than error on every reconcile.
func newProviderRegistry(mgr ctrl.Manager) *gitops.Registry {
	argocdProvider := &argocdprovider.Provider{Client: mgr.GetClient(), ControlPlaneNamespace: "argocd"}
	registry := gitops.NewRegistry(
		argocdProvider,
		&fluxprovider.Provider{Client: mgr.GetClient()},
	)
	if apiprobe.KindAvailable(mgr.GetRESTMapper(), kargoprovider.StageGVK) {
		registry.Register(&kargoprovider.Provider{Client: mgr.GetClient(), ArgoCD: argocdProvider})
	}
	return registry
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
