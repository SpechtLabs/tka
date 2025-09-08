// Package tailscale provides a Tailscale-network-only HTTP server by combining
// pkg/lnhttp with Tailscale's tsnet as the listener provider. This creates HTTP
// servers that are only accessible via the Tailscale network (tailnet), providing
// automatic security, TLS certificates, and identity resolution.
//
// The package builds on pkg/lnhttp to provide:
//   - Network Isolation: HTTP server only accessible via Tailscale network
//   - Automatic TLS: HTTPS certificates handled by Tailscale
//   - Identity Resolution: Built-in user identity and capability checking
//   - Funnel Detection: Ability to detect and reject public Funnel traffic
//   - Standard Interface: Drop-in replacement for http.Server
//
// Example usage:
//
//	// Create server with Tailscale networking
//	server := tailscale.NewServer("myapp",
//		tailscale.WithPort(443),
//		tailscale.WithDebug(true),
//	)
//
//	// Define handler with identity checking
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		// Reject Funnel traffic
//		if tailscale.IsFunnelRequest(r) {
//			http.Error(w, "Access denied", http.StatusForbidden)
//			return
//		}
//
//		// Get user identity
//		info, err := server.WhoIs(r.Context(), r.RemoteAddr)
//		if err != nil {
//			http.Error(w, "Authentication failed", http.StatusUnauthorized)
//			return
//		}
//
//		fmt.Fprintf(w, "Hello, %s!", info.LoginName)
//	})
//
//	// Start server
//	if err := server.Serve(ctx, handler); err != nil {
//		log.Fatal(err)
//	}
//
// For detailed documentation and examples, see:
// https://spechtlabs.github.io/tka/reference/developer/tailscale-server
package tailscale

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/lnhttp"
	"go.uber.org/zap"

	// Tailscale
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// Server provides an HTTP server that is only accessible via Tailscale network.
// It embeds lnhttp.Server and adds Tailscale-specific functionality including
// automatic TLS, identity resolution, and network isolation.
//
// The server uses Tailscale's tsnet package to create a listener that is only
// accessible from within the tailnet, providing automatic security and avoiding
// the need for public ingress or complex firewall configurations.
type Server struct {
	*lnhttp.Server

	// Configuration options
	debug bool
	port  int

	// stateDir specifies the directory to use for Tailscale state storage.
	// If empty, a directory is selected automatically under os.UserConfigDir
	// based on the name of the binary.
	//
	// If you want to use multiple tsnet services in the same binary, you will
	// need to make sure that stateDir is set uniquely for each service. A good
	// pattern is to have a "base" directory and append the hostname.
	stateDir string
	hostname string

	// Tailscale components
	ts        *tsnet.Server    // Embedded Tailscale server
	lc        *local.Client    // Local client for WhoIs lookups
	st        *ipnstate.Status // Connection status
	serverURL string           // Full server URL (e.g., "https://myapp.tailnet.ts.net:443")
}

// NewServer creates a new Tailscale HTTP server with the given hostname and options.
//
// The hostname parameter specifies the Tailscale hostname for this server (e.g., "myapp").
// The server will be accessible at https://hostname.tailnet.ts.net (or the configured port).
//
// Configuration options can be provided to customize the server behavior:
//   - WithPort: Set the listening port (default: 443)
//   - WithDebug: Enable debug logging
//   - WithStateDir: Set Tailscale state directory
//   - HTTP timeout options: WithReadTimeout, WithWriteTimeout, etc.
//
// Example:
//
//	server := tailscale.NewServer("myapp",
//		tailscale.WithPort(443),
//		tailscale.WithStateDir("/var/lib/myapp/ts-state"),
//		tailscale.WithDebug(false),
//	)
//
// The returned server must be started with Serve() and can be gracefully
// stopped with Shutdown().
func NewServer(hostname string, opts ...Option) *Server {
	// Initialize Tailscale server early to pass into the listener provider
	ts := &tsnet.Server{Hostname: hostname}

	// Construct the underlying http.Server with sane defaults
	httpSrv := &http.Server{
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Create lnhttp server with tsnet provider
	lnSrv := lnhttp.NewServer(httpSrv, &tsnetListenerProvider{ts: ts})

	server := &Server{
		Server:    lnSrv,
		debug:     false,
		port:      443,
		stateDir:  "",
		hostname:  hostname,
		ts:        ts,
		lc:        nil,
		st:        nil,
		serverURL: "",
	}

	// Apply options AFTER lnhttp.Server is initialized so options can touch timeouts safely
	for _, opt := range opts {
		opt(server)
	}

	// Ensure Addr matches configured port if not explicitly set
	if strings.TrimSpace(server.Addr) == "" && server.port > 0 {
		server.Addr = fmt.Sprintf(":%d", server.port)
	}

	// Apply state dir after options
	server.ts.Dir = server.stateDir

	// Configure debug logging if needed
	if server.debug {
		server.ts.Logf = otelzap.L().Sugar().Debugf
	}

	// Ensure ConnContext stashes the connection for later inspection (e.g., Funnel detection)
	server.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, CtxConnKey{}, c)
	}

	return server
}

// Serve starts the Tailscale HTTP server with the provided handler.
//
// This method:
//  1. Connects to the Tailscale network using the configured hostname
//  2. Creates a tailnet-only listener using tsnet
//  3. Starts the HTTP server with the provided handler
//  4. Returns when the server stops (via Shutdown) or encounters an error
//
// The context is used for the initial Tailscale connection setup. Once connected,
// the server runs until Shutdown is called or an error occurs.
//
// The handler will receive requests from authenticated Tailscale devices. Use
// IsFunnelRequest() to detect and reject public traffic if needed.
//
// Example:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if tailscale.IsFunnelRequest(r) {
//			http.Error(w, "Access denied", http.StatusForbidden)
//			return
//		}
//		fmt.Fprintf(w, "Hello from tailnet!")
//	})
//
//	if err := server.Serve(ctx, handler); err != nil {
//		log.Printf("Server error: %v", err)
//	}
func (s *Server) Serve(ctx context.Context, handler http.Handler) humane.Error {
	if err := s.connectTailnet(ctx); err != nil {
		return humane.Wrap(err, "failed to connect to tailnet", "check (debug) logs for more details")
	}

	if err := s.Server.Serve(ctx, handler); err != nil {
		return humane.Wrap(err, "failed to serve HTTP")
	}

	return nil
}

// Shutdown gracefully shuts down the tailscale
func (s *Server) Shutdown(ctx context.Context) humane.Error {
	if s.Server != nil {
		if err := s.Server.Shutdown(ctx); err != nil {
			otelzap.L().Error("failed to shutdown HTTP server", zap.Error(err))
			return humane.Wrap(err, "failed to shutdown HTTP server")
		}
	}
	return nil
}

func (s *Server) connectTailnet(ctx context.Context) humane.Error {
	var err error
	s.st, err = s.ts.Up(ctx)
	if err != nil {
		return humane.Wrap(err, "failed to start api tailscale", "check (debug) logs for more details")
	}

	portSuffix := ""
	if s.port != 443 {
		portSuffix = fmt.Sprintf(":%d", s.port)
	}

	s.serverURL = fmt.Sprintf("https://%s%s", strings.TrimSuffix(s.st.Self.DNSName, "."), portSuffix)

	if s.lc, err = s.ts.LocalClient(); err != nil {
		return humane.Wrap(err, "failed to get local api client", "check (debug) logs for more details")
	}

	otelzap.L().InfoContext(ctx, "tka tailscale running", zap.String("url", s.serverURL))
	return nil
}

// ListenAndServe provides a stdlib-compatible method to serve over the Tailnet using the configured Addr or port.
func (s *Server) ListenAndServe() error {
	// Use background context for compatibility; prefer Serve(ctx, handler) in new code.
	if err := s.Serve(context.Background(), s.Handler); err != nil {
		return err.Cause()
	}
	return nil
}

// Identity returns a WhoIsFunc backed by this server's tailscale local client.
func (s *Server) Identity() WhoIsFunc {
	return s.WhoIs
}

// WhoIs resolves identity information for a remote address using the tailscale local client.
func (s *Server) WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, error) {
	who, err := s.lc.WhoIs(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	info := &WhoIsInfo{
		LoginName: who.UserProfile.LoginName,
		CapMap:    who.CapMap,
		IsTagged:  who.Node.View().IsTagged(),
	}
	return info, nil
}

// tsnetListenerProvider is the default ListenerProvider backed by tsnet.Server.
type tsnetListenerProvider struct {
	ts *tsnet.Server
}

func (p *tsnetListenerProvider) Listen(ctx context.Context, network string, address string) (net.Listener, error) {
	return p.ts.Listen(network, address)
}
