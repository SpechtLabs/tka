// Package lnhttp provides a thin abstraction layer over Go's standard http.Server
// that decouples listener creation from server operation. This enables pluggable
// listener implementations (like Tailscale's tsnet) while maintaining full
// compatibility with the standard HTTP server interface.
//
// The key abstraction is the ListenerProvider interface, which allows
// applications to inject custom listener implementations for testing,
// custom protocols, or specialized networking (like Tailscale's tsnet).
//
// Example usage:
//
//	// Create a custom provider
//	provider := &myListenerProvider{}
//
//	// Create HTTP server with custom timeouts
//	httpSrv := &http.Server{
//		ReadTimeout:  30 * time.Second,
//		WriteTimeout: 60 * time.Second,
//		Addr:         ":8080",
//	}
//
//	// Create lnhttp server
//	server := lnhttp.NewServer(httpSrv, provider)
//
//	// Start serving
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprintf(w, "Hello from lnhttp!")
//	})
//
//	if err := server.Serve(ctx, handler); err != nil {
//		log.Fatal(err)
//	}
//
// For more examples and detailed documentation, see:
// https://spechtlabs.github.io/tka/reference/developer/lnhttp-server
package lnhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ListenerProvider abstracts how the HTTP server obtains a net.Listener.
// This enables swapping custom listeners (e.g., tsnet, Unix sockets, in-memory pipes)
// or plain TCP listeners in tests.
//
// Implementations should handle the context appropriately - cancellation should
// abort listener creation, but the returned listener's lifetime is managed
// separately by the HTTP server.
//
// Example implementation:
//
//	type TCPProvider struct{}
//
//	func (p *TCPProvider) Listen(ctx context.Context, network, address string) (net.Listener, error) {
//		var lc net.ListenConfig
//		return lc.Listen(ctx, network, address)
//	}
type ListenerProvider interface {
	// Listen creates a listener for the given network and address.
	// The context is used for the listener creation process only;
	// the returned listener's lifetime is managed by the HTTP server.
	//
	// Common network values are "tcp", "tcp4", "tcp6", "unix", "unixpacket".
	// The address format depends on the network type.
	Listen(ctx context.Context, network string, address string) (net.Listener, error)
}

// Server is a thin wrapper around *http.Server that obtains its listener
// from a pluggable ListenerProvider. This allows the same HTTP server logic
// to work with different networking backends.
//
// The Server embeds *http.Server, so all standard HTTP server methods and
// fields are available. The key difference is that Listen* operations use
// the provided ListenerProvider instead of the standard net.Listen functions.
type Server struct {
	*http.Server

	// Provider supplies the network listener for this server.
	// Must not be nil when calling Serve.
	Provider ListenerProvider
}

// NewServer constructs a new lnhttp Server with an embedded http.Server and
// the given ListenerProvider.
//
// If the http.Server parameter is nil, a default server with no configuration
// is created. The provider parameter specifies how the server will obtain
// its network listener.
//
// Example:
//
//	httpSrv := &http.Server{
//		Addr:         ":8080",
//		ReadTimeout:  30 * time.Second,
//		WriteTimeout: 60 * time.Second,
//	}
//	server := lnhttp.NewServer(httpSrv, &myProvider{})
func NewServer(s *http.Server, provider ListenerProvider) *Server {
	if s == nil {
		s = &http.Server{}
	}
	return &Server{Server: s, Provider: provider}
}

// Serve starts the HTTP server using a listener obtained from Provider.
// The context is passed to Provider.Listen for listener creation; it does not
// control the server lifecycle (use Shutdown for graceful termination).
//
// The method will:
//  1. Call Provider.Listen to obtain a network listener
//  2. Set the provided handler on the embedded http.Server
//  3. Call http.Server.Serve with the obtained listener
//  4. Return when the server stops (normally via Shutdown)
//
// Returns nil when the server shuts down gracefully, or an error if
// listener creation or server operation fails.
//
// Example:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprintf(w, "Hello, World!")
//	})
//
//	ctx := context.Background()
//	if err := server.Serve(ctx, handler); err != nil {
//		log.Printf("Server error: %v", err)
//	}
func (s *Server) Serve(ctx context.Context, handler http.Handler) error {
	if s.Provider == nil {
		return fmt.Errorf("lnhttp: Provider is nil")
	}

	address := s.Addr
	if address == "" {
		address = ":http"
	}

	ln, err := s.Provider.Listen(ctx, "tcp", address)
	if err != nil {
		return err
	}

	s.Handler = handler
	if err := s.Server.Serve(ln); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

// ListenAndServe starts the server using the configured Addr and the
// embedded http.Server's Handler. This provides compatibility with the
// standard http.Server interface.
//
// This method uses context.Background() for listener creation. For more
// control over the listener creation context, use Serve instead.
//
// Example:
//
//	server.Handler = myHandler
//	if err := server.ListenAndServe(); err != nil {
//		log.Fatal(err)
//	}
func (s *Server) ListenAndServe() error {
	return s.Serve(context.Background(), s.Handler)
}

// Shutdown gracefully shuts down the server by delegating to the embedded
// http.Server's Shutdown method. This will close the listener and wait for
// existing connections to complete.
//
// The context controls the shutdown timeout. If the context expires before
// shutdown completes, Shutdown returns the context's error.
//
// Example:
//
//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := server.Shutdown(shutdownCtx); err != nil {
//		log.Printf("Shutdown error: %v", err)
//	}
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
