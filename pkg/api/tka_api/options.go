package tka_api

import (
	"github.com/spechtlabs/tka/pkg/auth"
	mw "github.com/spechtlabs/tka/pkg/middleware/auth"
	ts "github.com/spechtlabs/tka/pkg/tailscale"
)

// Option defines a function type used to modify the configuration of a TKAServer during its initialization.
type Option func(*TKAServer)

// WithDebug enables or disables debug mode for the TKAServer.
func WithDebug(enable bool) Option {
	return func(tka *TKAServer) {
		tka.debug = enable
	}
}

// WithRetryAfterSeconds configures the default Retry-After value used by 202 responses.
func WithRetryAfterSeconds(seconds int) Option {
	return func(tka *TKAServer) {
		if seconds > 0 {
			tka.retryAfterSeconds = seconds
		}
	}
}

// WithAuthService injects a custom AuthService implementation for the API handlers.
func WithAuthService(svc auth.Service) Option {
	return func(tka *TKAServer) {
		tka.auth = svc
	}
}

// WithAuthMiddleware injects a custom AuthMiddleware implementation for the API router.
func WithAuthMiddleware(mw mw.Middleware) Option {
	return func(tka *TKAServer) {
		tka.authMW = mw
	}
}

// WithTailnetServer injects the tailscale-backed server used to serve the HTTP API.
func WithTailnetServer(s *ts.Server) Option {
	return func(tka *TKAServer) {
		tka.tsServer = s
	}
}
