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

// Server provides a generic HTTP server that connects to a Tailnet
type Server struct {
	*lnhttp.Server

	// Options
	debug    bool
	port     int
	stateDir string
	hostname string

	// Tailscale Server
	ts        *tsnet.Server
	lc        *local.Client
	st        *ipnstate.Status
	serverURL string // "https://foo.bar.ts.net"
}

// NewServer creates a new Server with the given hostname and options
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

// Serve starts the Server with the provided HTTP handler
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
