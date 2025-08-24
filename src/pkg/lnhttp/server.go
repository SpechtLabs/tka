package lnhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ListenerProvider abstracts how the HTTP server obtains a net.Listener.
// This enables swapping custom listeners (e.g., tsnet) or plain TCP in tests.
type ListenerProvider interface {
	Listen(ctx context.Context, network string, address string) (net.Listener, error)
}

// Server is a thin wrapper around *http.Server that obtains its listener
// from a pluggable ListenerProvider.
type Server struct {
	*http.Server
	Provider ListenerProvider
}

// NewServer constructs a new lnhttp Server with an embedded http.Server.
func NewServer(s *http.Server, provider ListenerProvider) *Server {
	if s == nil {
		s = &http.Server{}
	}
	return &Server{Server: s, Provider: provider}
}

// Serve starts the HTTP server using a listener obtained from Provider.
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

// ListenAndServe starts the server using the configured Addr.
func (s *Server) ListenAndServe() error {
	return s.Serve(context.Background(), s.Handler)
}

// Shutdown delegates to the embedded http.Server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
