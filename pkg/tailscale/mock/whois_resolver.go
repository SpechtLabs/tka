package mock

import (
	"context"
	"fmt"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/tailscale"
)

type MockWhoIsResolverOption func(*MockWhoIsResolver)

func WithWhoIsResponse(remoteAddr string, info *tailscale.WhoIsInfo) MockWhoIsResolverOption {
	return func(m *MockWhoIsResolver) {
		resp, ok := m.responses[remoteAddr]
		if !ok {
			resp = mockWhoIsResolverResponse{info: nil, err: nil}
		}
		resp.info = info
		m.responses[remoteAddr] = resp
	}
}

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

func WithWhoIsResponses(responses map[string]mockWhoIsResolverResponse) MockWhoIsResolverOption {
	return func(m *MockWhoIsResolver) {
		m.responses = responses
	}
}

type mockWhoIsResolverResponse struct {
	info *tailscale.WhoIsInfo
	err  error
}

type MockWhoIsResolver struct {
	responses map[string]mockWhoIsResolverResponse
}

func NewMockWhoIsResolver(opts ...MockWhoIsResolverOption) tailscale.WhoIsResolver {
	m := &MockWhoIsResolver{
		responses: make(map[string]mockWhoIsResolverResponse),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *MockWhoIsResolver) WhoIs(ctx context.Context, remoteAddr string) (*tailscale.WhoIsInfo, humane.Error) {
	resp, ok := m.responses[remoteAddr]
	if !ok {
		return nil, humane.Wrap(fmt.Errorf("no response for remote address %s", remoteAddr), "no response for remote address", "check (debug) logs for more details")
	}

	return resp.info, humane.Wrap(resp.err, "error getting WhoIs", "check (debug) logs for more details")
}
