// Package mock provides mock implementations for testing tshttp functionality.
// This package contains mock implementations of the interfaces defined in the
// parent tshttp package, enabling comprehensive unit testing.
package mock

import (
	"context"
	"net"

	humane "github.com/sierrasoftworks/humane-errors-go"
	ts "github.com/spechtlabs/tka/pkg/tshttp"
	"tailscale.com/ipn/ipnstate"
)

// MockTSNet implements ts.TSNet for unit tests with enhanced testability.
// It provides configurable behavior for all TSNet methods and tracks method calls.
type MockTSNet struct {
	// Up method configuration
	UpStatus *ipnstate.Status // Status to return from Up()
	UpErr    error            // Error to return from Up()
	UpCalled bool             // Whether Up() was called

	// Listen method configuration
	ListenErr    error          // Error to return from Listen()
	TLSErr       error          // Error to return from ListenTLS()
	FunnelErr    error          // Error to return from ListenFunnel()
	ListenCalled map[string]int // Tracks calls by network type

	// WhoIs configuration
	Whois       ts.WhoIsResolver // WhoIsResolver to return from LocalWhoIs()
	WhoIsErr    error            // Error to return from LocalWhoIs()
	WhoIsCalled bool             // Whether LocalWhoIs() was called

	// Configuration tracking
	Dir  string               // Directory set via SetDir()
	Logf func(string, ...any) // Log function set via SetLogf()
}

// NewMockTSNet creates a new MockTSNet with sensible defaults for testing.
// The mock is configured with a "Running" backend state and a test hostname.
func NewMockTSNet() *MockTSNet {
	return &MockTSNet{
		ListenCalled: make(map[string]int),
		UpStatus: &ipnstate.Status{
			BackendState: "Running",
			Self: &ipnstate.PeerStatus{
				DNSName: "test-host.tailnet.ts.net.",
			},
		},
	}
}

// Up simulates connecting to the Tailscale control plane.
// It returns the configured UpStatus and UpErr, and marks UpCalled as true.
func (m *MockTSNet) Up(ctx context.Context) (*ipnstate.Status, error) {
	m.UpCalled = true
	return m.UpStatus, m.UpErr
}

// Listen simulates creating a network listener.
// It increments the call counter for the network type and returns a nopListener or ListenErr.
func (m *MockTSNet) Listen(network, addr string) (net.Listener, error) {
	m.ListenCalled[network]++
	return &nopListener{}, m.ListenErr
}

// ListenTLS simulates creating a TLS listener.
// It increments the "tls" call counter and returns a nopListener or TLSErr.
func (m *MockTSNet) ListenTLS(network, addr string) (net.Listener, error) {
	m.ListenCalled["tls"]++
	return &nopListener{}, m.TLSErr
}

// ListenFunnel simulates creating a Funnel listener.
// It increments the "funnel" call counter and returns a nopListener or FunnelErr.
func (m *MockTSNet) ListenFunnel(network, addr string) (net.Listener, error) {
	m.ListenCalled["funnel"]++
	return &nopListener{}, m.FunnelErr
}

// LocalWhoIs simulates getting a WhoIsResolver.
// It marks WhoIsCalled as true and returns the configured Whois resolver or WhoIsErr.
func (m *MockTSNet) LocalWhoIs() (ts.WhoIsResolver, error) {
	m.WhoIsCalled = true
	if m.WhoIsErr != nil {
		return nil, m.WhoIsErr
	}
	if m.Whois != nil {
		return m.Whois, nil
	}
	return &MockWhoIs{}, nil
}

// SetDir simulates setting the Tailscale state directory.
func (m *MockTSNet) SetDir(dir string) {
	m.Dir = dir
}

// SetLogf simulates setting the logging function.
func (m *MockTSNet) SetLogf(logf func(string, ...any)) {
	m.Logf = logf
}

// nopListener is a no-op net.Listener implementation for testing.
// It provides minimal functionality to satisfy the net.Listener interface.
type nopListener struct{}

// Accept always returns context.Canceled to simulate a cancelled listener.
func (nopListener) Accept() (net.Conn, error) { return nil, context.Canceled }

// Close always returns nil (successful close).
func (nopListener) Close() error { return nil }

// Addr returns a dummy IP address.
func (nopListener) Addr() net.Addr { return &net.IPAddr{} }

// MockWhoIs is a configurable WhoIsResolver implementation for tests.
// It can be configured to return specific responses or errors.
type MockWhoIs struct {
	Resp *ts.WhoIsInfo // Response to return from WhoIs()
	Err  error         // Error to return from WhoIs()
}

// WhoIs simulates a WhoIs lookup.
// It returns the configured Resp or Err, or a default test response.
func (m *MockWhoIs) WhoIs(ctx context.Context, remoteAddr string) (*ts.WhoIsInfo, humane.Error) {
	if m.Err != nil {
		return nil, humane.Wrap(m.Err, "mock error", "this is a test mock error")
	}
	if m.Resp != nil {
		return m.Resp, nil
	}
	// Default response for testing
	return &ts.WhoIsInfo{
		LoginName: "test@example.com",
		Tags:      []string{},
	}, nil
}
