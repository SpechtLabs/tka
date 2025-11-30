package auth

import (
	"github.com/spechtlabs/tka/pkg/tsnet"
)

// Option is a functional option for configuring the Gin authentication middleware.
type Option[capRule tsnet.TailscaleCapability] func(*ginAuthMiddleware[capRule])

// AllowFunnelRequest returns an Option that configures whether Tailscale Funnel
// requests are allowed through the authentication middleware.
func AllowFunnelRequest[capRule tsnet.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowFunnel = allowed
	}
}

// AllowTaggedNodes returns an Option that configures whether requests from
// Tailscale tagged nodes (as opposed to user-owned nodes) are allowed.
func AllowTaggedNodes[capRule tsnet.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowTagged = allowed
	}
}
