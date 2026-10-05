package api

import (
	"github.com/prometheus/client_golang/prometheus"
	humane "github.com/sierrasoftworks/humane-errors-go"
)

// loginAttempts tracks login attempts by cluster role and outcome
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

// RegisterMetrics registers the API's metrics with reg. The server registers them
// once at startup with the default registry, which its metrics endpoint serves.
func RegisterMetrics(reg prometheus.Registerer) humane.Error {
	if err := reg.Register(loginAttempts); err != nil {
		return humane.Wrap(err, "failed to register the API metrics", "register them once, before the server starts")
	}

	return nil
}
