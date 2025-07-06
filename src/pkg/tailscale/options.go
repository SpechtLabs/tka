package tailscale

import "tailscale.com/tailcfg"

// Option defines a function type used to modify the configuration of a TKAServer during its initialization.
type Option func(*TKAServer)

// WithDebug enables or disables debug mode for the TKAServer.
func WithDebug(enable bool) Option {
	return func(tka *TKAServer) {
		tka.debug = enable
	}
}

// WithPort returns an Option that sets the port for the TKAServer. (Default: 443)
func WithPort(port int) Option {
	return func(tka *TKAServer) {
		tka.port = port
	}
}

// WithStateDir sets the state directory for the tsnet state
//
// The dir parameter specifies the name of the directory to use for
// state. If empty, a directory is selected automatically
// under os.UserConfigDir (https://golang.org/pkg/os/#UserConfigDir).
// based on the name of the binary.
//
// If you want to use multiple tsnet services in the same
// binary, you will need to make sure that Dir is set uniquely
// for each service. A good pattern for this is to have a
// "base" directory (such as your mutable storag
func WithStateDir(dir string) Option {
	return func(tka *TKAServer) {
		tka.stateDir = dir
	}
}

// WithPeerCapName sets the `capName` field of a `TKAServer` to the provided `tailcfg.PeerCapability`.
func WithPeerCapName(capName tailcfg.PeerCapability) Option {
	return func(tka *TKAServer) {
		tka.capName = capName
	}
}
