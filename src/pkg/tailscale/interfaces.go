package tailscale

import (
	"context"
	"net"

	"tailscale.com/tailcfg"
)

// ListenerProvider abstracts how the HTTP server obtains a net.Listener.
// This enables swapping tsnet for a standard TCP listener or a test double.
type ListenerProvider interface {
	Listen(ctx context.Context, network string, address string) (net.Listener, error)
}

// WhoIsInfo captures the subset of identity information needed by middleware.
type WhoIsInfo struct {
	LoginName string
	CapMap    tailcfg.PeerCapMap
	IsTagged  bool
}

// WhoIsResolver provides identity lookups for a remote address.
// Implementations typically wrap Tailscale's local client.
type WhoIsResolver interface {
	WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, error)
}
