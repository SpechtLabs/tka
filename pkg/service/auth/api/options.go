package api

import (
	mw "github.com/spechtlabs/tka/pkg/middleware"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

// Option defines a functional option pattern for configuring TKAServer instances.
// Options are applied during NewTKAServer() construction to customize server behavior.
// This pattern allows for flexible, readable server configuration without complex constructors.
//
// Example usage:
//
//	server, err := NewTKAServer(tailscaleServer, nil,
//	  WithDebug(true),
//	  WithRetryAfterSeconds(10),
//	  WithAuthMiddleware(mockAuth),
//	)
type Option func(*TKAServer)

// WithRetryAfterSeconds configures the Retry-After header value for asynchronous operations.
// This affects HTTP 202 (Accepted) responses when credentials are being provisioned.
// The value tells clients how long to wait before polling for completion.
func WithRetryAfterSeconds(seconds int) Option {
	return func(tka *TKAServer) {
		if seconds > 0 {
			tka.retryAfterSeconds = seconds
		}
	}
}

// WithAuthMiddleware replaces the default Tailscale authentication middleware.
// This is primarily used for testing with mock authentication or for custom
// authentication implementations.
func WithAuthMiddleware(m mw.Middleware) Option {
	return func(tka *TKAServer) {
		tka.authMiddleware = m
	}
}

// WithPrometheusMiddleware replaces the default Prometheus middleware.
// This is primarily used for testing with mock Prometheus or for custom
// Prometheus implementations.
func WithPrometheusMiddleware(p *ginprometheus.Prometheus) Option {
	return func(tka *TKAServer) {
		tka.sharedPrometheus = p
	}
}
