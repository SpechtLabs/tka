package operator

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var reconcilerDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "tka_reconciler_duration",
		Help: "How long the reconcile loop ran for in microseconds",
	},
	[]string{
		"reconciler",
		"name",
		"namespace",
	},
)

// userSignInsTotal tracks the total number of successful user sign-ins per cluster role
var userSignInsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tka_user_signins_total",
		Help: "Total number of successful user sign-ins by cluster role",
	},
	[]string{
		"cluster_role",
		"username",
	},
)

// activeUserSessions tracks the current number of active user sessions per cluster role
var activeUserSessions = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "tka_active_user_sessions",
		Help: "Current number of active user sessions by cluster role",
	},
	[]string{
		"cluster_role",
	},
)

func init() {
	metrics.Registry.MustRegister(reconcilerDuration)
	metrics.Registry.MustRegister(userSignInsTotal)
	metrics.Registry.MustRegister(activeUserSessions)
}
