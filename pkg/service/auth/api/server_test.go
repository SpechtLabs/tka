package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/spechtlabs/tka/pkg/client/k8s/mock"
	mwMock "github.com/spechtlabs/tka/pkg/middleware/auth/mock"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/auth/api"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newTestServer(t *testing.T, auth k8s.TkaClient, rule capability.Rule) (*api.TKAServer, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: rule, OmitRule: rule.Role == "" && rule.Period == ""}

	srv, err := api.NewTKAServer(nil, nil,
		api.WithAuthMiddleware(authMwMock),
	)
	require.NoError(t, err)

	if err := srv.LoadApiRoutes(auth); err != nil {
		t.Fatalf("failed to load api routes: %v", err)
	}

	ts := httptest.NewServer(srv.Engine())
	t.Cleanup(ts.Close)
	return srv, ts
}

func doReq(t *testing.T, ts *httptest.Server, method, path string, headers map[string]string, body any) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func requireErrorMessage(t *testing.T, body []byte, want string) {
	t.Helper()
	var er models.ErrorResponse
	require.NoError(t, json.Unmarshal(body, &er), string(body))
	require.Equal(t, want, er.Message)
}

var missingError = humane.Wrap(k8serrors.NewNotFound(schema.GroupResource{Group: "x", Resource: "y"}, "name"), "missing")
var noSigninError = humane.Wrap(k8serrors.NewNotFound(schema.GroupResource{Group: "x", Resource: "y"}, "name"), "no signin")

func TestNewTKAServer_RoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}
	s, herr := api.NewTKAServer(nil, nil, api.WithAuthMiddleware(authMwMock))
	require.NoError(t, herr)
	require.NotNil(t, s)
	require.NotNil(t, s.Engine())

	authSvc := mock.NewMockTkaClient()

	require.Error(t, s.LoadApiRoutes(nil))
	require.NoError(t, s.LoadApiRoutes(authSvc))

	// Maps the key to if the route is expected
	expected := map[string]struct {
		Expected bool
		Seen     bool
	}{
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:     {Expected: true, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:      {Expected: true, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.KubeconfigApiRoute: {Expected: true, Seen: false},
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LogoutApiRoute:    {Expected: true, Seen: false},
		http.MethodGet + " /orchestrator/v1alpha1/clusters":                  {Expected: false, Seen: false},
		http.MethodPost + " /orchestrator/v1alpha1/clusters":                 {Expected: false, Seen: false},
		http.MethodGet + " /metrics/controller":                              {Expected: true, Seen: false},
		http.MethodGet + " /swagger":                                         {Expected: true, Seen: false},
	}

	for _, r := range s.Engine().Routes() {
		key := r.Method + " " + r.Path
		if _, ok := expected[key]; ok {
			status := expected[key]
			status.Seen = true
			expected[key] = status
		}
	}

	for route, status := range expected {
		if status.Expected && !status.Seen {
			t.Errorf("missing route %s", route)
		}
		if !status.Expected && status.Seen {
			t.Errorf("unexpected route %s", route)
		}
	}
}

func TestNewTKAServer_SwaggerRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}
	s, herr := api.NewTKAServer(nil, nil,
		api.WithAuthMiddleware(authMwMock),
		api.WithRetryAfterSeconds(10),
		api.WithDebug(true),
	)
	require.NoError(t, herr)
	require.NotNil(t, s)
	require.NotNil(t, s.Engine())

	// Create a test server for making HTTP requests
	ts := httptest.NewServer(s.Engine())
	defer ts.Close()

	resp, _ := doReq(t, ts, http.MethodGet, "/swagger", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, http.StatusMovedPermanently, resp.Request.Response.StatusCode)
	require.Equal(t, "/swagger/index.html", resp.Request.Response.Header.Get("Location"))
}

func TestTKAServer_Serve(t *testing.T) {
	tests := []struct {
		name          string
		expectError   bool
		errorContains []string
	}{
		{
			name:          "no tailscale server configured",
			expectError:   true,
			errorContains: []string{"tailscale server not configured"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}

			// Create server without tailscale server (nil)
			s, herr := api.NewTKAServer(nil, nil, api.WithAuthMiddleware(authMwMock))
			require.NoError(t, herr)
			require.NotNil(t, s)

			// Call Serve
			ctx := context.Background()
			err := s.Serve(ctx)

			// Check error expectations
			if tt.expectError {
				require.Error(t, err)
				for _, contains := range tt.errorContains {
					require.Contains(t, err.Error(), contains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTKAServer_Shutdown(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "no tailscale server configured",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}

			// Create server without tailscale server (nil)
			s, herr := api.NewTKAServer(nil, nil, api.WithAuthMiddleware(authMwMock))
			require.NoError(t, herr)
			require.NotNil(t, s)

			// Call Shutdown
			ctx := context.Background()
			err := s.Shutdown(ctx)

			// Check error expectations
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
