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

func WithReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.ReadTimeout = timeout
		}
	}
}

func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.ReadHeaderTimeout = timeout
		}
	}
}

func WithIdleTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.IdleTimeout = timeout
		}
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		if s.Server != nil {
			s.WriteTimeout = timeout
		}
	}
}
