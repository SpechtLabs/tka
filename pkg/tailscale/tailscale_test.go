package tailscale_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spechtlabs/tka/pkg/tailscale/mock"
	"github.com/stretchr/testify/require"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

func TestNewServer(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name         string
		hostname     string
		opts         []tailscale.Option
		wantAddr     string
		wantTimeouts map[string]time.Duration
	}{
		{
			name:     "default configuration",
			hostname: "app",
			opts:     nil,
			wantAddr: ":443",
			wantTimeouts: map[string]time.Duration{
				"read":       10 * time.Second,
				"readHeader": 5 * time.Second,
				"write":      20 * time.Second,
				"idle":       120 * time.Second,
			},
		},
		{
			name:     "custom port and timeouts",
			hostname: "custom",
			opts: []tailscale.Option{
				tailscale.WithPort(8080),
				tailscale.WithReadTimeout(15 * time.Second),
				tailscale.WithWriteTimeout(25 * time.Second),
			},
			wantAddr: ":8080",
			wantTimeouts: map[string]time.Duration{
				"read":  15 * time.Second,
				"write": 25 * time.Second,
			},
		},
		{
			name:     "debug and state dir options",
			hostname: "debug-app",
			opts: []tailscale.Option{
				tailscale.WithDebug(true),
				tailscale.WithStateDir("/tmp/test-state"),
			},
			wantAddr: ":443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			s := tailscale.NewServer(tt.hostname, tt.opts...)

			require.NotNil(t, s.Server)
			require.Equal(t, tt.wantAddr, s.Addr)

			if timeout, ok := tt.wantTimeouts["read"]; ok {
				require.Equal(t, timeout, s.ReadTimeout)
			}
			if timeout, ok := tt.wantTimeouts["readHeader"]; ok {
				require.Equal(t, timeout, s.ReadHeaderTimeout)
			}
			if timeout, ok := tt.wantTimeouts["write"]; ok {
				require.Equal(t, timeout, s.WriteTimeout)
			}
			if timeout, ok := tt.wantTimeouts["idle"]; ok {
				require.Equal(t, timeout, s.IdleTimeout)
			}

			// Server should be shutdownable without being started
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			require.NoError(t, s.Shutdown(ctx))
		})
	}
}

func TestServer_Start(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*testing.T, *mock.MockTSNet)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successful start",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				// Default mock setup is successful
			},
			wantErr: false,
		},
		{
			name: "up fails",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.UpErr = errors.New("connection failed")
			},
			wantErr: true,
			errMsg:  "failed to start api tailscale",
		},
		{
			name: "whois setup fails",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.WhoIsErr = errors.New("whois failed")
			},
			wantErr: true,
			errMsg:  "failed to get local api client",
		},
		{
			name: "idempotent start",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				// Default mock setup is successful
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			tt.setupMock(t, mockTS)

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))
			ctx := context.Background()

			err := s.Start(ctx)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.True(t, mockTS.UpCalled)
			require.True(t, mockTS.WhoIsCalled)

			// Test idempotent behavior
			if tt.name == "idempotent start" {
				err2 := s.Start(ctx)
				require.NoError(t, err2)
			}
		})
	}
}

func TestServer_ConnectionState(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name          string
		setupMock     func(*testing.T, *mock.MockTSNet)
		port          int
		started       bool
		wantErr       bool
		errMsg        string
		wantConnected bool
		wantState     string
	}{
		{
			name:          "initial state - not started",
			started:       false,
			wantConnected: false,
			wantState:     "NoState",
		},
		{
			name:    "successful connection with default port",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.UpStatus = &ipnstate.Status{
					BackendState: "Running",
					Self: &ipnstate.PeerStatus{
						DNSName: "myapp.tailnet.ts.net.",
					},
				}
			},
			port:          443,
			wantConnected: true,
			wantState:     "Running",
		},
		{
			name:    "successful connection with custom port",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.UpStatus = &ipnstate.Status{
					BackendState: "Running",
					Self: &ipnstate.PeerStatus{
						DNSName: "myapp.tailnet.ts.net.",
					},
				}
			},
			port:          8080,
			wantConnected: true,
			wantState:     "Running",
		},
		{
			name:    "stopped state",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.UpStatus.BackendState = "Stopped"
			},
			wantConnected: false,
			wantState:     "Stopped",
		},
		{
			name:    "up fails",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.UpErr = errors.New("tailscale up failed")
			},
			wantErr: true,
			errMsg:  "failed to start api tailscale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			if tt.setupMock != nil {
				tt.setupMock(t, mockTS)
			}

			opts := []tailscale.Option{tailscale.WithTSNet(mockTS)}
			if tt.port != 0 {
				opts = append(opts, tailscale.WithPort(tt.port))
			}

			s := tailscale.NewServer("myapp", opts...)
			ctx := context.Background()

			if tt.started {
				err := s.Start(ctx)
				if tt.wantErr {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.errMsg)
					return
				}
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantConnected, s.IsConnected())
			require.Equal(t, tt.wantState, s.BackendState())
		})
	}
}

func TestServer_ListenMethods(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		setupMock func(*testing.T, *mock.MockTSNet)
		started   bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "ListenTCP requires start",
			method:  "tcp",
			started: false,
			wantErr: true,
			errMsg:  "Start() first",
		},
		{
			name:    "ListenTLS requires start",
			method:  "tls",
			started: false,
			wantErr: true,
			errMsg:  "Start() first",
		},
		{
			name:    "ListenFunnel requires start",
			method:  "funnel",
			started: false,
			wantErr: true,
			errMsg:  "Start() first",
		},
		{
			name:    "Listen requires start",
			method:  "listen",
			started: false,
			wantErr: true,
			errMsg:  "Start() first",
		},
		{
			name:    "ListenTCP success",
			method:  "tcp",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				// Default mock is successful
			},
			wantErr: false,
		},
		{
			name:    "ListenTLS success",
			method:  "tls",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				// Default mock is successful
			},
			wantErr: false,
		},
		{
			name:    "ListenFunnel success",
			method:  "funnel",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				// Default mock is successful
			},
			wantErr: false,
		},
		{
			name:    "ListenTCP error",
			method:  "tcp",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.ListenErr = errors.New("listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create tcp listener",
		},
		{
			name:    "ListenTLS error",
			method:  "tls",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.TLSErr = errors.New("tls listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create TLS listener",
		},
		{
			name:    "ListenFunnel error",
			method:  "funnel",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.FunnelErr = errors.New("funnel listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create Funnel listener",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			if tt.setupMock != nil {
				tt.setupMock(t, mockTS)
			}

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))

			if tt.started {
				err := s.Start(context.Background())
				require.NoError(t, err)
			}

			var listener net.Listener
			var err error

			switch tt.method {
			case "tcp":
				listener, err = s.ListenTCP(":8080")
			case "tls":
				listener, err = s.ListenTLS(":8080")
			case "funnel":
				listener, err = s.ListenFunnel(":8080")
			case "listen":
				listener, err = s.Listen("tcp", ":8080")
			}

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				require.Nil(t, listener)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, listener)

			// Verify the mock was called appropriately
			if tt.started {
				switch tt.method {
				case "tcp", "listen":
					require.Equal(t, 1, mockTS.ListenCalled["tcp"])
				case "tls":
					require.Equal(t, 1, mockTS.ListenCalled["tls"])
				case "funnel":
					require.Equal(t, 1, mockTS.ListenCalled["funnel"])
				}
			}
		})
	}
}

func TestServer_WhoIs(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		started   bool
		setupMock func(*testing.T, *mock.MockTSNet)
		addr      string
		wantErr   bool
		errMsg    string
		wantInfo  *tailscale.WhoIsInfo
	}{
		{
			name:    "requires start",
			started: false,
			addr:    "100.100.100.100:443",
			wantErr: true,
			errMsg:  "WhoIs resolver not available",
		},
		{
			name:    "successful lookup",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.Whois = &mock.MockWhoIs{
					Resp: &tailscale.WhoIsInfo{
						LoginName: "alice@example.com",
						Tags:      []string{},
					},
				}
			},
			addr:    "100.100.100.100:443",
			wantErr: false,
			wantInfo: &tailscale.WhoIsInfo{
				LoginName: "alice@example.com",
				Tags:      []string{},
			},
		},
		{
			name:    "whois error",
			started: true,
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.Whois = &mock.MockWhoIs{
					Err: errors.New("lookup failed"),
				}
			},
			addr:    "100.100.100.100:443",
			wantErr: true,
			errMsg:  "mock error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			if tt.setupMock != nil {
				tt.setupMock(t, mockTS)
			}

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))

			if tt.started {
				err := s.Start(context.Background())
				require.NoError(t, err)
			}

			info, err := s.WhoIs(context.Background(), tt.addr)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				require.Nil(t, info)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantInfo, info)
		})
	}
}

func TestServer_ServeNetworks(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		network   string
		setupMock func(*testing.T, *mock.MockTSNet)
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "tcp network",
			network: "tcp",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.ListenErr = errors.New("listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "tls network",
			network: "tls",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "funnel network",
			network: "funnel",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.FunnelErr = errors.New("funnel failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name:    "custom network",
			network: "custom",
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
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
			tt.setupMock(t, mockTS)

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

func TestServer_HighLevelMethods(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name      string
		testFn    func(*testing.T, *tailscale.Server) error
		setupMock func(*testing.T, *mock.MockTSNet)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "ListenAndServe",
			testFn: func(t *testing.T, s *tailscale.Server) error {
				t.Helper()
				return s.ListenAndServe()
			},
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.ListenErr = errors.New("listen failed")
			},
			wantErr: true,
			errMsg:  "failed to create listener",
		},
		{
			name: "ListenAndServeTLS",
			testFn: func(t *testing.T, s *tailscale.Server) error {
				t.Helper()
				return s.ListenAndServeTLS("", "")
			},
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to serve TLS",
		},
		{
			name: "ListenAndServeFunnel",
			testFn: func(t *testing.T, s *tailscale.Server) error {
				t.Helper()
				return s.ListenAndServeFunnel()
			},
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.FunnelErr = errors.New("funnel failed")
			},
			wantErr: true,
			errMsg:  "failed to serve Funnel",
		},
		{
			name: "ServeTLS",
			testFn: func(t *testing.T, s *tailscale.Server) error {
				t.Helper()
				return s.ServeTLS(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
				m.TLSErr = errors.New("tls failed")
			},
			wantErr: true,
			errMsg:  "failed to serve TLS",
		},
		{
			name: "ServeFunnel",
			testFn: func(t *testing.T, s *tailscale.Server) error {
				t.Helper()
				return s.ServeFunnel(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			setupMock: func(t *testing.T, m *mock.MockTSNet) {
				t.Helper()
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
			tt.setupMock(t, mockTS)

			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))
			s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			err := tt.testFn(t, s)

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestServer_Shutdown(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name    string
		started bool
	}{
		{
			name:    "shutdown without start",
			started: false,
		},
		{
			name:    "shutdown after start",
			started: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			mockTS := mock.NewMockTSNet()
			s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))

			if tt.started {
				err := s.Start(context.Background())
				require.NoError(t, err)
			}

			// First shutdown
			err1 := s.Shutdown(context.Background())
			require.NoError(t, err1)

			// Second shutdown (idempotent)
			err2 := s.Shutdown(context.Background())
			require.NoError(t, err2)

			// Stop should also work
			err3 := s.Stop(context.Background())
			require.NoError(t, err3)
		})
	}
}

func TestServer_HTTPCompatibility(t *testing.T) {
	t.Helper()
	t.Parallel()

	s := tailscale.NewServer("test")

	// Should embed http.Server
	require.NotNil(t, s.Server)

	// Should allow direct property modification
	s.ReadTimeout = 30 * time.Second
	require.Equal(t, 30*time.Second, s.ReadTimeout)

	// Should allow handler assignment
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s.Handler = handler
	require.NotNil(t, s.Handler)
}

func TestServer_HandlerAssignment(t *testing.T) {
	t.Helper()
	t.Parallel()

	t.Run("handler assignment", func(t *testing.T) {
		t.Helper()
		s := tailscale.NewServer("test")

		// Initially no handler
		require.Nil(t, s.Handler)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})

		// Set handler directly
		s.Handler = handler
		require.NotNil(t, s.Handler)

		// Test handler functionality
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		s.Handler.ServeHTTP(rr, req)

		require.Equal(t, http.StatusTeapot, rr.Code)
	})

	t.Run("serve does not set handler when listener fails", func(t *testing.T) {
		t.Helper()
		mockTS := mock.NewMockTSNet()
		mockTS.ListenErr = errors.New("listen failed")

		s := tailscale.NewServer("test", tailscale.WithTSNet(mockTS))
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		require.Nil(t, s.Handler)

		err := s.Serve(context.Background(), handler, "tcp")
		require.Error(t, err)

		// Handler should still be nil because listener creation failed
		require.Nil(t, s.Handler)
	})
}

// Test helper types for IsFunnelRequest tests
type fakeTLSWrapper struct{ inner net.Conn }

func (f *fakeTLSWrapper) NetConn() net.Conn { return f.inner }

type nopConn struct{}

func (nopConn) Read(b []byte) (int, error)         { return 0, nil }
func (nopConn) Write(b []byte) (int, error)        { return len(b), nil }
func (nopConn) Close() error                       { return nil }
func (nopConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (nopConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (nopConn) SetDeadline(t time.Time) error      { return nil }
func (nopConn) SetReadDeadline(t time.Time) error  { return nil }
func (nopConn) SetWriteDeadline(t time.Time) error { return nil }

func TestIsFunnelRequest(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name       string
		headers    map[string]string
		ctxConn    any
		wantFunnel bool
	}{
		{
			name:       "header indicates funnel",
			headers:    map[string]string{"Tailscale-Funnel-Request": "1"},
			wantFunnel: true,
		},
		{
			name:       "funnel conn in context",
			ctxConn:    &ipn.FunnelConn{},
			wantFunnel: true,
		},
		{
			name:       "tls-wrapped funnel conn",
			ctxConn:    &fakeTLSWrapper{inner: &ipn.FunnelConn{}},
			wantFunnel: true,
		},
		{
			name:       "regular connection",
			ctxConn:    nopConn{},
			wantFunnel: false,
		},
		{
			name:       "no connection info",
			wantFunnel: false,
		},
		{
			name:       "nil context connection",
			ctxConn:    nil,
			wantFunnel: false,
		},
		{
			name:       "both header and connection indicate funnel",
			headers:    map[string]string{"Tailscale-Funnel-Request": "1"},
			ctxConn:    &ipn.FunnelConn{},
			wantFunnel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			req := httptest.NewRequest(http.MethodGet, "http://example", nil)

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if tt.ctxConn != nil {
				ctx := context.WithValue(req.Context(), tailscale.CtxConnKey{}, tt.ctxConn)
				req = req.WithContext(ctx)
			}

			got := tailscale.IsFunnelRequest(req)
			require.Equal(t, tt.wantFunnel, got)
		})
	}
}

func TestWhoIsInfo_IsTagged(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want bool
	}{
		{name: "nil tags", tags: nil, want: false},
		{name: "empty tags", tags: []string{}, want: false},
		{name: "has tags", tags: []string{"tag:svc"}, want: true},
		{name: "multiple tags", tags: []string{"tag:svc", "tag:prod"}, want: true},
		{name: "single tag", tags: []string{"tag:web"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			info := tailscale.WhoIsInfo{Tags: tt.tags}
			require.Equal(t, tt.want, info.IsTagged())
		})
	}
}

func TestWhoIsInfo_Fields(t *testing.T) {
	t.Helper()
	t.Parallel()

	info := tailscale.WhoIsInfo{
		LoginName: "alice@example.com",
		Tags:      []string{"tag:web", "tag:prod"},
	}

	require.Equal(t, "alice@example.com", info.LoginName)
	require.Equal(t, []string{"tag:web", "tag:prod"}, info.Tags)
	require.True(t, info.IsTagged())
}

func TestServer_AddressHandling(t *testing.T) {
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
