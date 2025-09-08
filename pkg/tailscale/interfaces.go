package tailscale

import (
	"context"

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

	// IsTagged indicates whether the source connection is from a tagged device.
	// Tagged devices represent service accounts rather than human users.
	IsTagged bool
}

// WhoIsFunc is a function that resolves identity information for a remote address.
// Implementations typically use Tailscale's local client to perform WhoIs lookups
// on the provided remote address.
//
// Example usage:
//
//	info, err := whoIsFunc(ctx, request.RemoteAddr)
//	if err != nil {
//		// Handle authentication failure
//		return fmt.Errorf("identity lookup failed: %w", err)
//	}
//
//	if info.IsTagged {
//		// Handle service account differently
//		return handleServiceAccount(info)
//	}
//
//	// Check for required capability
//	if caps, ok := info.CapMap["example.com/cap/admin"]; ok {
//		// User has admin capabilities
//		return handleAdminRequest(info, caps)
//	}
type WhoIsFunc func(ctx context.Context, remoteAddr string) (*WhoIsInfo, error)
