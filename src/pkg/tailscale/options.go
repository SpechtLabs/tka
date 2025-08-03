package tailscale

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
