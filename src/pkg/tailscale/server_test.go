package tailscale_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/stretchr/testify/require"
)

// dummyConn is a minimal net.Conn implementation for testing ConnContext.
type dummyConn struct{}

func (d *dummyConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (d *dummyConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (d *dummyConn) Close() error                       { return nil }
func (d *dummyConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (d *dummyConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (d *dummyConn) SetDeadline(t time.Time) error      { return nil }
func (d *dummyConn) SetReadDeadline(t time.Time) error  { return nil }
func (d *dummyConn) SetWriteDeadline(t time.Time) error { return nil }

func TestServer_Init(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		opts       []tailscale.Option
		wantAddr   string
		checkConn  bool
		checkIdent bool
		checkPlain bool
	}{
		{
			name:       "with port sets Addr and conn context, identity available",
			opts:       []tailscale.Option{tailscale.WithPort(8123)},
			wantAddr:   ":8123",
			checkConn:  true,
			checkIdent: true,
			checkPlain: true,
		},
		{
			name:       "default port sets Addr to :443, still sets conn context and identity",
			opts:       nil,
			wantAddr:   ":443",
			checkConn:  true,
			checkIdent: true,
			checkPlain: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tailscale.NewServer("unit-test-host", tc.opts...)

			require.NotNil(t, s.Server, "expected embedded server to be initialized")
			require.Equal(t, tc.wantAddr, s.Addr)

			if tc.checkConn {
				require.NotNil(t, s.ConnContext, "expected ConnContext to be set")
				ctx := s.ConnContext(context.Background(), &dummyConn{})
				require.NotNil(t, ctx.Value(tailscale.CtxConnKey{}), "expected ConnContext to store connection in context")
			}

			if tc.checkIdent {
				require.NotNil(t, s.Identity(), "expected Identity to return a non-nil function")
			}

			if tc.checkPlain {
				// a plain context should not already contain a connection value
				require.Nil(t, context.Background().Value(tailscale.CtxConnKey{}))
			}
		})
	}
}

func TestServer_Shutdown_WithoutRunningServer(t *testing.T) {
	t.Parallel()

	s := tailscale.NewServer("unit-test-host")
	// server not started; Shutdown should return nil
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(shutdownCtx))
}
