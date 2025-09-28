package auth

import "github.com/spechtlabs/tka/pkg/tshttp"

type Option[capRule tshttp.TailscaleCapability] func(*ginAuthMiddleware[capRule])

func AllowFunnelRequest[capRule tshttp.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowFunnel = allowed
	}
}

func AllowTaggedNodes[capRule tshttp.TailscaleCapability](allowed bool) Option[capRule] {
	return func(m *ginAuthMiddleware[capRule]) {
		m.allowTagged = allowed
	}
}
