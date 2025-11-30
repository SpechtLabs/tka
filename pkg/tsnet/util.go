package tsnet

import (
	"net"
	"net/http"

	"tailscale.com/ipn"
)

// CtxConnKey is a context key used to store a net.Conn in an HTTP request's context.
// This allows handlers to access the underlying network connection for advanced
// inspection or connection-specific operations such as Funnel detection.
type CtxConnKey struct{}

// IsFunnelRequest reports whether an HTTP request is coming over Tailscale Funnel.
//
// Tailscale Funnel allows public internet access to your tailnet services,
// but you may want to reject such traffic for security-sensitive operations.
// This function detects Funnel traffic by checking for Funnel-specific headers
// and connection types.
//
// Returns true if the request came through Tailscale Funnel (public internet),
// false if it came directly through the tailnet.
//
// Always check for Funnel requests in authentication-sensitive handlers,
// as Funnel traffic bypasses Tailscale's device authentication.
func IsFunnelRequest(r *http.Request) bool {
	// If we're funneling through the local tailscaled, it will set this HTTP
	// header.
	if r.Header.Get("Tailscale-Funnel-Request") != "" {
		return true
	}

	// If the funneled connection is from tsnet, then the net.Conn will be of
	// type ipn.FunnelConn.
	netConn := r.Context().Value(CtxConnKey{})

	// If the connection is wrapped (e.g. by TLS), unwrap it using a generic
	// interface that matches tls.Conn's NetConn() method. This improves
	// testability while preserving behavior for real tls.Conn.
	type netConner interface{ NetConn() net.Conn }
	if wrapper, ok := netConn.(netConner); ok {
		netConn = wrapper.NetConn()
	}

	if _, ok := netConn.(*ipn.FunnelConn); ok {
		return true
	}

	return false
}
