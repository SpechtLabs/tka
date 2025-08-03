package tailscale

// TailscaleOption defines a function type for modifying Server configuration
type TailscaleOption func(*Server)

// WithDebug sets debug mode for the Server
func WithDebug(debug bool) TailscaleOption {
	return func(s *Server) {
		s.debug = debug
	}
}

// WithPort sets the listening port for the Server
func WithPort(port int) TailscaleOption {
	return func(s *Server) {
		s.port = port
	}
}

// WithStateDir sets the state directory for Tailscale
func WithStateDir(dir string) TailscaleOption {
	return func(s *Server) {
		s.stateDir = dir
	}
}
