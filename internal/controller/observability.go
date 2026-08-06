package controller

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	eventWaitingForSchedule   = "WaitingForSchedule"
	eventWaitingForGitOpsSync = "WaitingForGitOpsSync"
	eventWaitingForRollout    = "WaitingForRollout"
	eventKickStarted          = "KickStarted"
	eventKickSucceeded        = "KickSucceeded"
	eventKickNoLongerRequired = "KickNoLongerRequired"
	eventKickFailed           = "KickFailed"
	eventOwnerUnknown         = "OwnerUnknown"
	eventOwnerAmbiguous       = "OwnerAmbiguous"
)

var (
	metricsOnce sync.Once

	kickRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kick_requests_total",
			Help: "Total number of completed KickRequest outcomes by provider and result.",
		},
		[]string{"provider", "result"},
	)
	kickRestartsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kick_restarts_total",
			Help: "Total number of restart execution attempts and outcomes by provider and result.",
		},
		[]string{"provider", "result"},
	)
	kickControllerErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kick_controller_errors_total",
			Help: "Total number of controller errors by controller and reason.",
		},
		[]string{"controller", "reason"},
	)
)

func registerControllerMetrics() {
	metricsOnce.Do(func() {
		ctrlmetrics.Registry.MustRegister(kickRequestsTotal, kickRestartsTotal, kickControllerErrorsTotal)
	})
}

func observeRequestResult(provider, result string) {
	kickRequestsTotal.WithLabelValues(normalizeProvider(provider), result).Inc()
}

func observeRestartResult(provider, result string) {
	kickRestartsTotal.WithLabelValues(normalizeProvider(provider), result).Inc()
}

func observeControllerError(controller, reason string) {
	if reason == "" {
		reason = "Unknown"
	}
	kickControllerErrorsTotal.WithLabelValues(controller, reason).Inc()
}

func normalizeProvider(provider string) string {
	if provider == "" {
		return "unknown"
	}
	return provider
}
