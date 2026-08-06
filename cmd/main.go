package main

import (
	"context"
	"flag"
	"os"
	"time"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"github.com/corewire/kick/internal/controller"
	"github.com/corewire/kick/internal/dependency"
	"github.com/corewire/kick/internal/executor"
	"github.com/corewire/kick/internal/freshness"
	"github.com/corewire/kick/internal/gitops"
	argocdprovider "github.com/corewire/kick/internal/gitops/argocd"
	"github.com/corewire/kick/internal/kickrequest"
	"github.com/corewire/kick/internal/observation"
	"github.com/corewire/kick/internal/policy"
	"github.com/corewire/kick/internal/rollout"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint address.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe address.")
	flag.BoolVar(&leaderElection, "leader-elect", false, "Enable leader election.")
	zapOptions := zap.Options{Development: true}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

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
	policyMatcher := &policy.DeploymentPolicyMatcher{Client: mgr.GetClient()}
	providerRegistry := gitops.NewRegistry(&argocdprovider.Provider{Client: mgr.GetClient(), ControlPlaneNamespace: "argocd"})
	if err := (&controller.KickRequestReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		PolicyMatcher:      policyMatcher,
		GateResolver:       &controller.RegistryGateResolver{Registry: providerRegistry},
		ObservationStore:   observation.NewLeaseStore(mgr.GetClient()),
		FreshnessEvaluator: &freshness.Evaluator{Inspector: &rollout.LiveRolloutInspector{Client: mgr.GetClient()}},
		RestartExecutor:    executor.NewRestartExecutor(mgr.GetClient(), 10*time.Minute),
		RequeueInterval:    30 * time.Second,
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}
	if err := (&controller.SourceObservationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Observer: observation.NewObserver(observation.NewLeaseStore(mgr.GetClient()), nil),
		Enqueuer: &controller.KickRequestEnqueuer{Client: mgr.GetClient(), Coalescer: kickrequest.NewCoalescer(mgr.GetClient(), kickrequest.RetentionConfig{}), PolicyMatcher: policyMatcher},
	}).SetupWithManager(mgr); err != nil {
		os.Exit(1)
	}
	if err := dependency.RegisterDeploymentReverseIndexes(context.Background(), mgr.GetFieldIndexer()); err != nil {
		os.Exit(1)
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
