package tailscale

import (
	"crypto/tls"
	"net/http"

	"tailscale.com/ipn"
)

// CtxConnKey is a context key used to store a net.Conn in an HTTP request's context.
// This allows handlers to access the underlying network connection for advanced
// inspection or connection-specific operations.
//
// Exported for tests and integration points that need to retrieve the connection.
//
// Example usage:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		if conn := r.Context().Value(tailscale.CtxConnKey{}); conn != nil {
//			if netConn, ok := conn.(net.Conn); ok {
//				fmt.Printf("Remote addr: %s", netConn.RemoteAddr())
//			}
//		}
//	}
type CtxConnKey struct{}

// IsFunnelRequest checks if an HTTP request is coming over Tailscale Funnel.
//
// Tailscale Funnel allows public internet access to your tailnet services,
// but you may want to reject such traffic for security-sensitive operations.
// This function detects Funnel traffic by checking for Funnel-specific headers
// and connection types.
//
// Returns true if the request came through Tailscale Funnel (public internet),
// false if it came directly through the tailnet.
//
// Example usage:
//
//	func authHandler(w http.ResponseWriter, r *http.Request) {
//		if tailscale.IsFunnelRequest(r) {
//			http.Error(w, "Access denied: Funnel requests not allowed",
//				http.StatusForbidden)
//			return
//		}
//		// Handle authenticated tailnet request
//	}
//
// Security note: Always check for Funnel requests in authentication-sensitive
// handlers, as Funnel traffic bypasses Tailscale's device authentication.
func IsFunnelRequest(r *http.Request) bool {
	// If we're funneling through the local tailscaled, it will set this HTTP
	// header.
	if r.Header.Get("Tailscale-Funnel-Request") != "" {
		return true
	}

	// If the funneled connection is from tsnet, then the net.Conn will be of
	// type ipn.FunnelConn.
	netConn := r.Context().Value(CtxConnKey{})

	// if the conn is wrapped inside TLS, unwrap it
	if tlsConn, ok := netConn.(*tls.Conn); ok {
		netConn = tlsConn.NetConn()
	}

	if _, ok := netConn.(*ipn.FunnelConn); ok {
		return true
	}

	return false
}
