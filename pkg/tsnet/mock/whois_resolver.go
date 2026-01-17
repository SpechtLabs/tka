package mock

import (
	"context"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/tsnet"
)

// MockWhoIsResolverOption is a functional option for configuring MockWhoIsResolver.
type MockWhoIsResolverOption func(*MockWhoIsResolver)

// WithWhoIsResponse configures a mock response for a specific remote address.
func WithWhoIsResponse(remoteAddr string, info *tsnet.WhoIsInfo) MockWhoIsResolverOption {
	return func(m *MockWhoIsResolver) {
		resp, ok := m.responses[remoteAddr]
		if !ok {
			resp = mockWhoIsResolverResponse{info: nil, err: nil}
		}
		resp.info = info
		m.responses[remoteAddr] = resp
	}
}

// WithWhoIsError configures an error response for a specific remote address.
func WithWhoIsError(remoteAddr string, err error) MockWhoIsResolverOption {
	return func(m *MockWhoIsResolver) {
		resp, ok := m.responses[remoteAddr]
		if !ok {
			resp = mockWhoIsResolverResponse{info: nil, err: nil}
		}
		resp.err = err
		m.responses[remoteAddr] = resp
	}
}

// WithWhoIsResponses configures multiple mock responses at once.
func WithWhoIsResponses(responses map[string]mockWhoIsResolverResponse) MockWhoIsResolverOption {
	return func(m *MockWhoIsResolver) {
		m.responses = responses
	}
}

type mockWhoIsResolverResponse struct {
	info *tsnet.WhoIsInfo
	err  error
}

// Compile-time interface verification
var _ tsnet.WhoIsResolver = &MockWhoIsResolver{}

// MockWhoIsResolver is a configurable mock implementation of tsnet.WhoIsResolver for testing.
type MockWhoIsResolver struct {
	responses map[string]mockWhoIsResolverResponse
}

// NewMockWhoIsResolver creates a new MockWhoIsResolver with the provided options.
func NewMockWhoIsResolver(opts ...MockWhoIsResolverOption) tsnet.WhoIsResolver {
	m := &MockWhoIsResolver{
		responses: make(map[string]mockWhoIsResolverResponse),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *MockWhoIsResolver) WhoIs(ctx context.Context, remoteAddr string) (*tsnet.WhoIsInfo, humane.Error) {
	resp, ok := m.responses[remoteAddr]
	if !ok {
		return nil, humane.New("no response for remote address "+remoteAddr, "check (debug) logs for more details")
	}

	if resp.err != nil {
		return resp.info, humane.Wrap(resp.err, "error getting WhoIs", "check (debug) logs for more details")
	}
	return resp.info, nil
}
