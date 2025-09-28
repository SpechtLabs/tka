// Package tshttp provides configuration options for HTTP servers on Tailscale networks.
// This file contains functional options for customizing server behavior.
package tshttp

import (
	"fmt"
	"time"
)

// Option defines a functional option for configuring a Server.
// Options are applied during server creation to customize behavior.
type Option func(*Server)

// WithDebug enables or disables debug logging for the Server.
// When enabled, detailed Tailscale operation logs will be output.
func WithDebug(debug bool) Option {
	return func(s *Server) {
		s.debug = debug
	}
}

// WithPort sets the listening port for the Server.
// The default port is 443. Setting a different port will affect the server URL.
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
		if s.Server != nil {
			s.Addr = fmt.Sprintf(":%d", port)
		}
	}
}

// WithStateDir sets the state directory for Tailscale
// If empty, a directory is selected automatically
// under os.UserConfigDir (https://golang.org/pkg/os/#UserConfigDir).
// based on the name of the binary.
//
// If you want to use multiple tsnet services in the same
// binary, you will need to make sure that Dir is set uniquely
// for each service. A good pattern for this is to have a
// "base" directory (such as your mutable storage folder) and
// then append the hostname on the end of it.
func WithStateDir(dir string) Option {
	return func(s *Server) {
		s.stateDir = dir
	}
}

// WithReadTimeout sets the maximum duration for reading the entire request,
// including the body. A zero or negative value means there will be no timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.ReadTimeout = timeout
		}
	}
}

// WithReadHeaderTimeout sets the amount of time allowed to read request headers.
// If zero, the value of ReadTimeout is used.
func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.ReadHeaderTimeout = timeout
		}
	}
}

// WithIdleTimeout sets the maximum amount of time to wait for the next request
// when keep-alives are enabled. If zero, the value of ReadTimeout is used.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.IdleTimeout = timeout
		}
	}
}

// WithWriteTimeout sets the maximum duration before timing out writes of the response.
// A zero or negative value means there will be no timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.WriteTimeout = timeout
		}
	}
}

// WithWhoIsResolver sets a custom WhoIsResolver for identity lookups.
// This is primarily useful for testing with mock resolvers.
func WithWhoIsResolver(whois WhoIsResolver) Option {
	return func(s *Server) {
		s.whois = whois
	}
}

// WithTSNet sets a custom TSNet implementation.
// This is primarily useful for testing with mock implementations.
func WithTSNet(ts TSNet) Option {
	return func(s *Server) {
		s.ts = ts
	}
}
