package lnhttp_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/pkg/lnhttp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type pipeListener struct{ connCh chan net.Conn }

func newPipeListener(t testing.TB) *pipeListener {
	t.Helper()
	return &pipeListener{connCh: make(chan net.Conn, 1)}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	c, ok := <-l.connCh
	if !ok {
		return nil, &net.OpError{Op: "accept", Err: context.Canceled}
	}
	return c, nil
}

func (l *pipeListener) Close() error   { close(l.connCh); return nil }
func (l *pipeListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4zero, Port: 0} }

type pipeProvider struct{ l *pipeListener }

func (p *pipeProvider) Listen(_ context.Context, _ string, _ string) (net.Listener, error) {
	return p.l, nil
}

func newPipeProvider(t testing.TB) (*pipeProvider, func(testing.TB) net.Conn) {
	t.Helper()
	pl := newPipeListener(t)
	prov := &pipeProvider{l: pl}
	connect := func(tb testing.TB) net.Conn {
		tb.Helper()
		client, server := net.Pipe()
		go func() { pl.connCh <- server }()
		return client
	}
	return prov, connect
}

type mockHandler struct{ mock.Mock }

func newMockHandler(t testing.TB) *mockHandler {
	t.Helper()
	return &mockHandler{}
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
	w.WriteHeader(http.StatusOK)
}

func writeRequestAndReadStatus(t testing.TB, c net.Conn) (int, error) {
	t.Helper()
	// Write minimal HTTP/1.1 request
	if _, err := c.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n")); err != nil {
		return 0, err
	}
	// Read status line
	r := bufio.NewReader(c)
	line, err := r.ReadString('\n')
	if err != nil {
		return 0, err
	}
	// Expect "HTTP/1.1 200 ..."
	if len(line) >= 12 {
		// parse status code at bytes 9-12 (space then 3 digits)
		var code int
		_, err = fmt.Sscanf(line, "HTTP/1.1 %d", &code)
		return code, err
	}
	return 0, fmt.Errorf("bad status line: %q", line)
}

func TestServer_Handler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		start     func(testing.TB, *lnhttp.Server, http.Handler) (chan error, func())
		provider  func(testing.TB) (*pipeProvider, func(testing.TB) net.Conn)
		setupHTTP func(*http.Server)
	}{
		{
			name: "Serve calls provided handler",
			start: func(tb testing.TB, s *lnhttp.Server, h http.Handler) (chan error, func()) {
				tb.Helper()
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan error, 1)
				go func() { done <- s.Serve(ctx, h) }()
				return done, func() { cancel() }
			},
			provider: newPipeProvider,
		},
		{
			name: "ListenAndServe uses embedded http.Server handler",
			start: func(tb testing.TB, s *lnhttp.Server, _ http.Handler) (chan error, func()) {
				tb.Helper()
				done := make(chan error, 1)
				go func() { done <- s.ListenAndServe() }()
				return done, func() {}
			},
			provider: newPipeProvider,
			setupHTTP: func(hs *http.Server) {
				// handler will be set to mock by caller
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prov, connect := tc.provider(t)
			httpSrv := &http.Server{}
			srv := lnhttp.NewServer(httpSrv, prov)

			mh := newMockHandler(t)
			mh.On("ServeHTTP", mock.Anything, mock.Anything).Once()

			if tc.setupHTTP != nil {
				// set mock as embedded handler for ListenAndServe case
				httpSrv.Handler = mh
				tc.setupHTTP(httpSrv)
			}

			done, cancel := tc.start(t, srv, mh)
			defer cancel()

			client := connect(t)
			defer client.Close()

			// write request and read status so server completes the request
			code, err := writeRequestAndReadStatus(t, client)
			require.NoError(t, err)
			require.Equal(t, 200, code)

			// ensure mock observed a call
			mh.AssertExpectations(t)

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
			defer shutdownCancel()
			require.NoError(t, srv.Shutdown(shutdownCtx))
			require.NoError(t, <-done)
		})
	}
}

func TestServer_Serve_Error(t *testing.T) {
	t.Parallel()
	srv := lnhttp.NewServer(&http.Server{}, nil)
	err := srv.Serve(context.Background(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	require.Error(t, err)
}
