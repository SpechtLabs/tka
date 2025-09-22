package auth

import (
	"github.com/spechtlabs/tka/pkg/tailscale"
)

type Option[capRule tailscale.TailscaleCapability] func(*ginAuthMiddleware[capRule])

func AllowFunnelRequest[capRule tailscale.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowFunnel = allowed
	}
}

func AllowTaggedNodes[capRule tailscale.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowTagged = allowed
	}
}
