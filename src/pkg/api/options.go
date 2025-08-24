package api

import "tailscale.com/tailcfg"

// Option defines a function type used to modify the configuration of a TKAServer during its initialization.
type Option func(*TKAServer)

// WithDebug enables or disables debug mode for the TKAServer.
func WithDebug(enable bool) Option {
	return func(tka *TKAServer) {
		tka.debug = enable
	}
}

// WithPeerCapName sets the `capName` field of a `TKAServer` to the provided `tailcfg.PeerCapability`.
func WithPeerCapName(capName tailcfg.PeerCapability) Option {
	return func(tka *TKAServer) {
		tka.capName = capName
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
