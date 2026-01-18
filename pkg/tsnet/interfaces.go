package tsnet

import (
	"context"
	"net"

	"github.com/sierrasoftworks/humane-errors-go"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

// WhoIsInfo captures identity information for a Tailscale client connection.
// This information is obtained via Tailscale's WhoIs API and can be used
// for authentication and authorization decisions.
type WhoIsInfo struct {
	// LoginName is the user's login name (e.g., "alice@example.com").
	LoginName string

	// CapMap contains the capability grants from Tailscale ACL policy.
	// Keys are capability names, values are the granted capabilities.
	CapMap tailcfg.PeerCapMap

	// Tags contains the Tailscale ACL tags assigned to this device.
	// Tagged devices represent service accounts rather than human users.
	Tags []string
}

// IsTagged indicates whether the source connection is from a tagged device.
// Tagged devices represent service accounts rather than human users.
func (w *WhoIsInfo) IsTagged() bool {
	return len(w.Tags) > 0
}

// WhoIsResolver resolves identity information for remote addresses.
// Implementations typically use Tailscale's local client to perform WhoIs lookups
// on the provided remote address.
//
// Example usage:
//
//	info, err := whoIsResolver.WhoIs(ctx, request.RemoteAddr)
//	if err != nil {
//		// Handle authentication failure
//		return fmt.Errorf("identity lookup failed: %w", err)
//	}
//
//	if info.IsTagged() {
//		// Handle service account differently
//		return handleServiceAccount(info)
//	}
type WhoIsResolver interface {
	// WhoIs resolves identity information for the given remote address.
	// The remoteAddr should be in the format "ip:port".
	WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, humane.Error)
}

// TailscaleCapability represents a capability that can be granted to a user.
// Capabilities are used for fine-grained access control within Tailscale ACLs.
type TailscaleCapability interface {
	// Priority returns the priority level of this capability.
	// Higher values indicate higher priority.
	Priority() int
}

// TSNet abstracts the subset of tsnet.Server functionality we use.
// This interface enables testing by allowing mock implementations.
type TSNet interface {
	// Up connects to the Tailscale control plane and returns the connection status.
	Up(ctx context.Context) (*ipnstate.Status, error)

	// Listen creates a listener on the Tailscale network for the given network and address.
	Listen(network, addr string) (net.Listener, error)
	// ListenTLS creates a TLS listener on the Tailscale network.
	ListenTLS(network, addr string) (net.Listener, error)
	// ListenFunnel creates a Funnel listener that accepts public internet traffic.
	ListenFunnel(network, addr string) (net.Listener, error)

	// LocalWhoIs returns a WhoIsResolver for identity lookups.
	LocalWhoIs() (WhoIsResolver, error)

	// SetDir sets the directory for Tailscale state storage.
	SetDir(dir string)
	// SetLogf sets the logging function for Tailscale operations.
	SetLogf(logf func(string, ...any))

	// Hostname returns the hostname of the Tailscale server.
	Hostname() string

	// GetPeerState returns the current peer state.
	GetPeerState() *ipnstate.PeerStatus

	// IsConnected returns true if the server is connected to the Tailscale network.
	IsConnected() bool

	// BackendState returns the current backend state.
	BackendState() BackendState
}
