package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

// loginAttempts tracks login attempts by cluster role and outcome.
var loginAttempts = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tka_login_attempts_total",
		Help: "Total number of login attempts by cluster role and outcome",
	},
	[]string{
		"username",
		"cluster_role",
		"outcome", // success, forbidden, error
	},
)

func init() {
	prometheus.MustRegister(loginAttempts)
}
