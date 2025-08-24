package tailscale

import (
	"fmt"
	"time"
)

// Option defines a function type for modifying Server configuration
type Option func(*Server)

// WithDebug sets debug mode for the Server
func WithDebug(debug bool) Option {
	return func(s *Server) {
		s.debug = debug
	}
}

// WithPort sets the listening port for the Server
func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
		if s.Server != nil && s.Server.Server != nil {
			s.Server.Addr = fmt.Sprintf(":%d", port)
		}
	}
}

// WithStateDir sets the state directory for Tailscale
func WithStateDir(dir string) Option {
	return func(s *Server) {
		s.stateDir = dir
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil && s.Server.Server != nil {
			s.Server.ReadTimeout = timeout
		}
	}
}

func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil && s.Server.Server != nil {
			s.Server.ReadHeaderTimeout = timeout
		}
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil && s.Server.Server != nil {
			s.Server.IdleTimeout = timeout
		}
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil && s.Server.Server != nil {
			s.Server.WriteTimeout = timeout
		}
	}
}
