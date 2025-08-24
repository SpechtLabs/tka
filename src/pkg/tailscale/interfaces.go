package tailscale

import (
	"context"

	"tailscale.com/tailcfg"
)

// WhoIsInfo captures the subset of identity information needed by middleware.
type WhoIsInfo struct {
	LoginName string
	CapMap    tailcfg.PeerCapMap
	IsTagged  bool
}

// WhoIsFunc is a function that resolves identity information for a remote address.
type WhoIsFunc func(ctx context.Context, remoteAddr string) (*WhoIsInfo, error)
