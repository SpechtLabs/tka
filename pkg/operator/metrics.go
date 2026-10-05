package operator

import (
	"github.com/prometheus/client_golang/prometheus"
	humane "github.com/sierrasoftworks/humane-errors-go"
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

// RegisterMetrics registers the operator's metrics with reg. The server registers
// them once at startup with controller-runtime's registry, which its metrics
// endpoint serves.
func RegisterMetrics(reg prometheus.Registerer) humane.Error {
	for _, c := range []prometheus.Collector{reconcilerDuration, userSignInsTotal, activeUserSessions} {
		if err := reg.Register(c); err != nil {
			return humane.Wrap(err, "failed to register the operator metrics", "register them once, before the operator starts")
		}
	}

	return nil
}
