package notify

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	deliveriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "kick_notification_deliveries_total",
		Help: "Notification webhook deliveries by outcome.",
	}, []string{"namespace", "policy", "outcome"})

	droppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kick_notification_dropped_total",
		Help: "Notification events discarded because the delivery queue was full.",
	})
)

func init() {
	metrics.Registry.MustRegister(deliveriesTotal, droppedTotal)
}

func observeDelivery(namespace, policy string, ok bool) {
	outcome := "failure"
	if ok {
		outcome = "success"
	}
	deliveriesTotal.WithLabelValues(namespace, policy, outcome).Inc()
}

func observeDropped() {
	droppedTotal.Inc()
}
