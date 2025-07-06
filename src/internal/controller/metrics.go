package controller

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

func init() {
	metrics.Registry.MustRegister(reconcilerDuration)
}
