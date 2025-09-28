package tailscale_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spechtlabs/tka/pkg/tailscale/mock"
	"github.com/stretchr/testify/require"
)

// TestServer_ServeNetworks_Integration tests the different network types for Serve method.
// This is an integration test that verifies the complete serve flow.
func TestServer_ServeNetworks_Integration(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		network   string
		setupMock func(*mock.MockTSNet)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "tcp network",
			network: "tcp",
			setupMock: func(m *mock.MockTSNet) {
				m.ListenErr = errors.New("listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "tls network",
			network: "tls",
			setupMock: func(m *mock.MockTSNet) {
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "funnel network",
			network: "funnel",
			setupMock: func(m *mock.MockTSNet) {
				m.FunnelErr = errors.New("funnel failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "custom network",
			network: "custom",
			setupMock: func(m *mock.MockTSNet) {
				m.ListenErr = errors.New("custom listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			tt.setupMock(mockTS)

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			err := s.Serve(context.Background(), handler, tt.network)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestServer_HighLevelMethods_Integration tests the convenience methods that wrap Serve.
// This is an integration test that verifies the complete serve flow for high-level methods.
func TestServer_HighLevelMethods_Integration(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		testFn    func(*tailscale.Server) error
		setupMock func(*mock.MockTSNet)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "ListenAndServe",
			testFn: func(s *tailscale.Server) error { return s.ListenAndServe() },
			setupMock: func(m *mock.MockTSNet) {
				m.ListenErr = errors.New("listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:   "ListenAndServeTLS",
			testFn: func(s *tailscale.Server) error { return s.ListenAndServeTLS("", "") },
			setupMock: func(m *mock.MockTSNet) {
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to serve TLS",
		},
		{
			name:   "ListenAndServeFunnel",
			testFn: func(s *tailscale.Server) error { return s.ListenAndServeFunnel() },
			setupMock: func(m *mock.MockTSNet) {
				m.FunnelErr = errors.New("funnel failed")
			},
			wantErr: true,
			errMsg:  "failed to serve Funnel",
		},
		{
			name: "ServeTLS",
			testFn: func(s *tailscale.Server) error {
				return s.ServeTLS(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			setupMock: func(m *mock.MockTSNet) {
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to serve TLS",
		},
		{
			name: "ServeFunnel",
			testFn: func(s *tailscale.Server) error {
				return s.ServeFunnel(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			setupMock: func(m *mock.MockTSNet) {
				m.FunnelErr = errors.New("funnel failed")
			},
			wantErr: true,
			errMsg:  "failed to serve Funnel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			tt.setupMock(mockTS)

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))
			s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			err := tt.testFn(s)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestServer_AddressHandling_Integration tests how the server handles different address configurations.
// This is an integration test that verifies address handling in a complete server setup.
func TestServer_AddressHandling_Integration(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name     string
		port     int
		wantAddr string
	}{
		{
			name:     "default port 443",
			port:     443,
			wantAddr: ":443",
		},
		{
			name:     "custom port 8080",
			port:     8080,
			wantAddr: ":8080",
		},
		{
			name:     "port 80",
			port:     80,
			wantAddr: ":80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			s := tailscale.NewServer("test",
				tailscale.WithTSNet(mockTS),
				tailscale.WithPort(tt.port),
			)

			require.Equal(t, tt.wantAddr, s.Addr)
		})
	}
}
