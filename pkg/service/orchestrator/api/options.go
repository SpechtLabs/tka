package api

import (
	mw "github.com/spechtlabs/tka/pkg/middleware"
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

// WithDebug configures debug mode for the TKAServer.
// When enabled, the server runs in Gin's debug mode with verbose logging.
// When disabled, the server runs in release mode for better performance.
func WithDebug(enable bool) Option {
	return func(tka *TKAServer) {
		tka.debug = enable
	}
}

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
