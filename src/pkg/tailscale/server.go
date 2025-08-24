package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"

	// Tailscale
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// Server provides a generic HTTP server that connects to a Tailnet
type Server struct {
	*http.Server

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

	// Abstractions
	listenerProvider ListenerProvider
	whoIs            WhoIsResolver
}

// NewServer creates a new Server with the given hostname and options
func NewServer(hostname string, opts ...Option) *Server {
	server := &Server{
		Server: &http.Server{
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      20 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
		debug:     false,
		port:      443,
		stateDir:  "",
		hostname:  hostname,
		ts:        nil,
		lc:        nil,
		st:        nil,
		serverURL: "",
	}

	// Apply options
	for _, opt := range opts {
		opt(server)
	}

	// Initialize Tailscale
	server.ts = &tsnet.Server{
		Hostname: hostname,
		Dir:      server.stateDir,
	}

	// Configure debug logging if needed
	if server.debug {
		server.ts.Logf = otelzap.L().Sugar().Debugf
	}

	// Ensure ConnContext stashes the connection for later inspection (e.g., Funnel detection)
	server.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, ctxConn{}, c)
	}

	return server
}

// Serve starts the Server with the provided HTTP handler
func (s *Server) Serve(ctx context.Context, handler http.Handler) humane.Error {
	if err := s.connectTailnet(ctx); err != nil {
		return humane.Wrap(err, "failed to connect to tailnet", "check (debug) logs for more details")
	}

	// Determine address
	address := s.Addr
	if strings.TrimSpace(address) == "" {
		address = fmt.Sprintf(":%d", s.port)
	}

	// Connect to the tailnet using the listener provider
	if s.listenerProvider == nil {
		s.listenerProvider = &tsnetListenerProvider{ts: s.ts}
	}

	ln, err := s.listenerProvider.Listen(ctx, "tcp", address)
	if err != nil {
		return humane.Wrap(err, "failed to listen on api network")
	}

	s.Handler = handler

	// Serve HTTP
	if err := s.Server.Serve(ln); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			otelzap.L().Info("tka tailscale stopped")
			return nil
		}

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

// GetServerURL returns the HTTPS URL for this tailscale
func (s *Server) GetServerURL() string {
	return s.serverURL
}

// LC returns the Tailscale local client
func (s *Server) LC() *local.Client {
	return s.lc
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

	// default WhoIs resolver
	if s.whoIs == nil {
		s.whoIs = &localWhoIsResolver{lc: s.lc}
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

// Identity returns the WhoIsResolver used by this server.
func (s *Server) Identity() WhoIsResolver {
	return s.whoIs
}

// tsnetListenerProvider is the default ListenerProvider backed by tsnet.Server.
type tsnetListenerProvider struct {
	ts *tsnet.Server
}

func (p *tsnetListenerProvider) Listen(ctx context.Context, network string, address string) (net.Listener, error) {
	return p.ts.Listen(network, address)
}

// localWhoIsResolver wraps a tailscale local client to provide WhoIs lookups.
type localWhoIsResolver struct {
	lc *local.Client
}

func (r *localWhoIsResolver) WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, error) {
	who, err := r.lc.WhoIs(ctx, remoteAddr)
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
