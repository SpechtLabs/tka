// Package tailscale provides a Tailscale-network-only HTTP server that creates HTTP
// servers that are only accessible via the Tailscale network (tailnet), providing
// automatic security, TLS certificates, and identity resolution.
//
// The package provides two usage patterns:
//
// # High-Level Usage (All-in-One)
//
// Use the Serve() method for a complete solution:
//
//	server := tailscale.NewServer("myapp",
//		tailscale.WithPort(443),
//		tailscale.WithDebug(true),
//	)
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		info, err := server.WhoIs(r.Context(), r.RemoteAddr)
//		if err != nil {
//			http.Error(w, "Authentication failed", http.StatusUnauthorized)
//			return
//		}
//		fmt.Fprintf(w, "Hello, %s!", info.LoginName)
//	})
//
//	if err := server.Serve(ctx, handler); err != nil {
//		log.Fatal(err)
//	}
//
// # Low-Level Usage
//
// Use Start() + ListenTCP() for more control with standard http.Server:
//
//	server := tailscale.NewServer("myapp")
//	if err := server.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//
//	listener, err := server.ListenTCP(":8080")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Use any http.Server - server is just for connection setup
//	httpServer := &http.Server{
//		Handler: myHandler,
//		ReadTimeout: 30 * time.Second,
//	}
//	go func() {
//		if err := httpServer.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
//			log.Printf("HTTP server error: %v", err)
//		}
//	}()
//
//	// Or use server directly as it IS an http.Server:
//	server.Handler = myHandler
//	go func() {
//		if err := server.Server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
//			log.Printf("HTTP server error: %v", err)
//		}
//	}()
//
//	// Shutdown
//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	if err := server.Shutdown(shutdownCtx); err != nil {
//		log.Printf("HTTP shutdown error: %v", err)
//	}
//	if err := server.Stop(shutdownCtx); err != nil {
//		log.Printf("Tailscale shutdown error: %v", err)
//	}
//
// For detailed documentation and examples, see:
// https://tka.specht-labs.de/reference/developer/tailscale-server
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

// Server provides an HTTP server that is only accessible via Tailscale network.
// It embeds http.Server directly, making it a true drop-in replacement that can be used
// in multiple ways:
//  1. High-level: Use Serve() for a complete HTTP server solution
//  2. Low-level: Use Start() + ListenTCP() + standard http.Server for more control
//  3. Standard: Use like a regular http.Server after calling Start()
//
// The server uses Tailscale's tsnet package to create listeners that are only
// accessible from within the tailnet, providing automatic security and avoiding
// the need for public ingress or complex firewall configurations.
type Server struct {
	// Embedded http.Server makes this a true drop-in replacement
	*http.Server

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
	started   bool             // Track if Start() has been called
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
	// Initialize Tailscale server
	ts := &tsnet.Server{Hostname: hostname}

	// Construct the underlying http.Server with sane defaults
	httpSrv := &http.Server{
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	server := &Server{
		Server:    httpSrv,
		debug:     false,
		port:      443,
		stateDir:  "",
		hostname:  hostname,
		ts:        ts,
		lc:        nil,
		st:        nil,
		serverURL: "",
		started:   false,
	}

	// Apply options AFTER http.Server is initialized so options can touch timeouts safely
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

// Start connects to the Tailscale network and prepares the server for accepting connections.
// This method separates connection setup from serving.
//
// After calling Start(), you can:
//   - Use ListenTCP() to get listeners for standard http.Server
//   - Use Serve() for a high-level all-in-one solution
//
// Example usage:
//
//	server := tailscale.NewServer("myapp")
//	if err := server.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//
//	listener, err := server.ListenTCP(":8080")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	httpServer := &http.Server{Handler: myHandler}
//	go httpServer.Serve(listener)
func (s *Server) Start(ctx context.Context) humane.Error {
	if s.started {
		return nil // Already started
	}

	if err := s.connectTailnet(ctx); err != nil {
		return err
	}

	s.started = true
	return nil
}

// ListenTCP creates a TCP listener on the Tailscale network.
// This method returns a standard net.Listener that can be used with any http.Server.
//
// The server must be started with Start() before calling this method.
//
// Example:
//
//	listener, err := server.ListenTCP(":8080")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	httpServer := &http.Server{
//		Handler: myHandler,
//		ReadTimeout: 30 * time.Second,
//	}
//	go httpServer.Serve(listener)
func (s *Server) ListenTCP(address string) (net.Listener, humane.Error) {
	if !s.started {
		return nil, humane.Wrap(fmt.Errorf("server not started"), "call Start() first")
	}
	listener, err := s.ts.Listen("tcp", address)
	if err != nil {
		return nil, humane.Wrap(err, "failed to create TCP listener")
	}
	return listener, nil
}

// Stop gracefully stops the Tailscale server.
//
// Example:
//
//	if err := server.Stop(ctx); err != nil {
//		log.Printf("Stop error: %v", err)
//	}
func (s *Server) Stop(ctx context.Context) humane.Error {
	s.started = false
	return s.Shutdown(ctx)
}

// Serve starts the Tailscale HTTP server with the provided handler.
// This is the high-level method that handles everything automatically.
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
	if !s.started {
		if err := s.Start(ctx); err != nil {
			return humane.Wrap(err, "failed to start server")
		}
	}

	// Get listener from tsnet
	address := s.Addr
	if address == "" {
		address = ":http"
	}

	listener, err := s.ts.Listen("tcp", address)
	if err != nil {
		return humane.Wrap(err, "failed to create listener")
	}

	// Set handler and serve
	s.Handler = handler
	if err := s.Server.Serve(listener); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return humane.Wrap(err, "failed to serve HTTP")
	}

	return nil
}

// Shutdown gracefully shuts down the tailscale server
func (s *Server) Shutdown(ctx context.Context) humane.Error {
	if s.Server != nil {
		if err := s.Server.Shutdown(ctx); err != nil {
			otelzap.L().Error("failed to shutdown HTTP server", zap.Error(err))
			return humane.Wrap(err, "failed to shutdown HTTP server")
		}
	}
	s.started = false
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

// WhoIs resolves identity information for a remote address using the tailscale local client.
func (s *Server) WhoIs(ctx context.Context, remoteAddr string) (*WhoIsInfo, humane.Error) {
	who, err := s.lc.WhoIs(ctx, remoteAddr)
	if err != nil {
		return nil, humane.Wrap(err, "failed to get WhoIs", "check (debug) logs for more details")
	}

	info := &WhoIsInfo{
		LoginName: who.UserProfile.LoginName,
		CapMap:    who.CapMap,
		Tags:      who.Node.Tags,
	}
	return info, nil
}
