package tailscale_test

import (
	"context"
	"testing"
	"time"

	"github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/stretchr/testify/require"
)

func TestServer_NewServer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		opts     []tailscale.Option
		hostname string
	}{
		{
			name:     "creates server with custom port",
			opts:     []tailscale.Option{tailscale.WithPort(8123)},
			hostname: "test-app",
		},
		{
			name:     "creates server with default settings",
			opts:     nil,
			hostname: "default-app",
		},
		{
			name:     "creates server with debug enabled",
			opts:     []tailscale.Option{tailscale.WithDebug(true)},
			hostname: "debug-app",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tailscale.NewServer(tc.hostname, tc.opts...)

			// Check that server is initialized
			require.NotNil(t, s, "expected server to be initialized")

			// Test that server can be shutdown without being started
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := s.Shutdown(shutdownCtx)
			require.NoError(t, err, "shutdown should succeed even if server not started")
		})
	}
}

func TestServer_CleanAPI(t *testing.T) {
	t.Parallel()

	t.Run("methods exist and return expected errors before Start", func(t *testing.T) {
		s := tailscale.NewServer("test-app")

		// ListenTCP should fail before Start() is called
		_, err := s.ListenTCP(":8080")
		require.Error(t, err, "ListenTCP should fail before Start")
		require.Contains(t, err.Error(), "Start() first", "error should mention calling Start() first")

		// Stop should succeed even if not started
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = s.Stop(ctx)
		require.NoError(t, err, "Stop should succeed even if not started")
	})
}

func TestServer_DropInReplacement(t *testing.T) {
	t.Parallel()

	t.Run("server embeds http.Server and can be used as such", func(t *testing.T) {
		s := tailscale.NewServer("test-app", tailscale.WithPort(8080))

		// Test that it embeds http.Server properly
		require.NotNil(t, s.Server, "should embed http.Server")

		// Test that we can set http.Server properties directly
		s.ReadTimeout = 30 * time.Second
		require.Equal(t, 30*time.Second, s.ReadTimeout)

		// Test that Addr is set correctly
		require.Equal(t, ":8080", s.Addr)

		// Test that Handler can be set
		s.Handler = nil // This should work without panic
		require.Nil(t, s.Handler)
	})
}

func TestServer_Shutdown_WithoutRunningServer(t *testing.T) {
	t.Parallel()

	s := tailscale.NewServer("unit-test-host")
	// server not started; Shutdown should return nil
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.Shutdown(shutdownCtx))
}
