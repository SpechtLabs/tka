package tailscale

import "time"

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
		s.srv.ReadTimeout = timeout
	}
}

func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.ReadHeaderTimeout = timeout
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.IdleTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.WriteTimeout = timeout
	}
}
