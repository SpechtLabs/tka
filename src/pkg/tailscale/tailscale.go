package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"

	// Tailscale
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

// Server provides a generic HTTP tailscale that connects to a Tailnet
type Server struct {
	// Options
	debug    bool
	port     int
	stateDir string

	// Tailscale Server
	ts        *tsnet.Server
	lc        *local.Client
	st        *ipnstate.Status
	serverURL string // "https://foo.bar.ts.net"

	// HTTP Server
	srv *http.Server
}

// NewServer creates a new Server with the given hostname and options
func NewServer(hostname string, opts ...Option) *Server {
	server := &Server{
		debug:     false,
		port:      443,
		stateDir:  "",
		ts:        nil,
		lc:        nil,
		st:        nil,
		serverURL: "",
		srv:       nil,
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

	return server
}

// Serve starts the Server with the provided HTTP handler
func (s *Server) Serve(ctx context.Context, handler http.Handler) humane.Error {
	if err := s.connectTailnet(ctx); err != nil {
		return humane.Wrap(err, "failed to connect to tailnet", "check (debug) logs for more details")
	}

	// Connect to the tailnet
	ln, err := s.ts.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return humane.Wrap(err, "failed to listen on api network")
	}

	// Get local client
	s.lc, err = s.ts.LocalClient()
	if err != nil {
		return humane.Wrap(err, "failed to get local client")
	}

	// Get status
	s.st, err = s.lc.Status(context.Background())
	if err != nil {
		return humane.Wrap(err, "failed to get status")
	}

	// Set tailscale URL
	if s.st.Self != nil && s.st.Self.DNSName != "" {
		s.serverURL = fmt.Sprintf("https://%s", s.st.Self.DNSName)
	}

	// Create and configure HTTP tailscale
	s.srv = &http.Server{
		Handler: handler,
	}

	// Serve HTTP
	if err := s.srv.Serve(ln); err != nil {
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
	if s.srv != nil {
		if err := s.srv.Shutdown(ctx); err != nil {
			otelzap.L().Error("failed to shutdown HTTP tailscale", zap.Error(err))
			return humane.Wrap(err, "failed to shutdown HTTP tailscale")
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

	otelzap.L().InfoContext(ctx, "tka tailscale running", zap.String("url", s.serverURL))
	return nil
}
