package api

import (
	"github.com/spechtlabs/tka/pkg/cluster"
	mw "github.com/spechtlabs/tka/pkg/middleware"
	"github.com/spechtlabs/tka/pkg/service"
	"github.com/spechtlabs/tka/pkg/service/models"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

// Option defines a functional option pattern for configuring TKAServer instances.
// Options are applied during NewTKAServer() construction to customize server behavior.
// This pattern allows for flexible, readable server configuration without complex constructors.
//
// Example usage:
//
//	server, err := NewTKAServer(tailscaleServer, nil,
//	  WithDebug(true),
//	  WithRetryAfterSeconds(10),
//	  WithAuthMiddleware(mockAuth),
//	)
type Option func(*TKAServer)

// WithRetryAfterSeconds configures the Retry-After header value for asynchronous operations.
// This affects HTTP 202 (Accepted) responses when credentials are being provisioned.
// The value tells clients how long to wait before polling for completion.
func WithRetryAfterSeconds(seconds int) Option {
	return func(tka *TKAServer) {
		if seconds > 0 {
			tka.retryAfterSeconds = seconds
		}
	}
}

// WithAuthMiddleware replaces the default Tailscale authentication middleware.
// This is primarily used for testing with mock authentication or for custom
// authentication implementations.
func WithAuthMiddleware(m mw.Middleware) Option {
	return func(tka *TKAServer) {
		tka.authMiddleware = m
	}
}

// WithPrometheusMiddleware replaces the default Prometheus middleware.
// This is primarily used for testing with mock Prometheus or for custom
// Prometheus implementations.
func WithPrometheusMiddleware(p *ginprometheus.Prometheus) Option {
	return func(tka *TKAServer) {
		tka.sharedPrometheus = p
	}
}

// WithClusterInfo configures the TKA server with cluster connection information.
// This information is exposed to authenticated users via the cluster-info API endpoint
// and is used by clients to configure their kubeconfig files for connecting to the cluster.
func WithClusterInfo(info *models.TkaClusterInfo) Option {
	return func(tka *TKAServer) {
		tka.clusterInfo = info
	}
}

// WithNewClusterInfo is a convenience function that creates and configures cluster information
// for the TKA server. This is useful when you want to construct the cluster info inline
// rather than creating a separate TkaClusterInfo struct.
func WithNewClusterInfo(serverURL string, caData string, labels map[string]string) Option {
	return func(tka *TKAServer) {
		tka.clusterInfo = &models.TkaClusterInfo{
			ServerURL:             serverURL,
			CAData:                caData,
			InsecureSkipTLSVerify: false, // Default to secure TLS verification
			Labels:                labels,
		}
	}
}

// WithGossipStore configures the TKA server with a gossip store for memberlist discovery.
func WithGossipStore(store cluster.GossipStore[service.NodeMetadata]) Option {
	return func(tka *TKAServer) {
		tka.gossipStore = store
	}
}
