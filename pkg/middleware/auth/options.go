package auth

import (
	"github.com/spechtlabs/tka/pkg/tsnet"
)

type Option[capRule tsnet.TailscaleCapability] func(*ginAuthMiddleware[capRule])

func AllowFunnelRequest[capRule tsnet.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowFunnel = allowed
	}
}

func AllowTaggedNodes[capRule tsnet.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowTagged = allowed
	}
}
