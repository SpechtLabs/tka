package tailscale

import (
	"context"

	"github.com/sierrasoftworks/humane-errors-go"
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

	Tags []string
}

// IsTagged indicates whether the source connection is from a tagged device.
// Tagged devices represent service accounts rather than human users.
func (w *WhoIsInfo) IsTagged() bool {
	return len(w.Tags) > 0
}

// WhoIsResolver is a function that resolves identity information for a remote address.
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
//	if info.IsTagged {
//		// Handle service account differently
//		return handleServiceAccount(info)
//	}
type WhoIsResolver interface {
	WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, humane.Error)
}

// TailscaleCapability is an interface that represents a capability that can be granted to a user.
type TailscaleCapability interface {
	Priority() int
}
