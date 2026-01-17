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
	"github.com/spechtlabs/tka/pkg/service/api"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/stretchr/testify/require"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var sharedPrometheus = ginprometheus.NewPrometheus("tka")

func newTestServer(t *testing.T, auth k8s.TkaClient, rule capability.Rule) (*api.TKAServer, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: rule, OmitRule: rule.Role == "" && rule.Period == ""}

	srv := api.NewTKAServer(
		api.WithAuthMiddleware(authMwMock),
		api.WithPrometheusMiddleware(sharedPrometheus),
	)

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
	req, err := http.NewRequestWithContext(context.Background(), method, ts.URL+path, rdr)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req) //nolint:golint-sl // DefaultClient is acceptable in tests
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

var missingError = humane.Wrap(k8serrors.NewNotFound(schema.GroupResource{Group: "x", Resource: "y"}, "name"), "missing", "verify the resource exists")
var noSigninError = humane.Wrap(k8serrors.NewNotFound(schema.GroupResource{Group: "x", Resource: "y"}, "name"), "no signin", "please sign in first")

func TestNewTKAServer_RoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}
	s := api.NewTKAServer(api.WithAuthMiddleware(authMwMock))
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
	s := api.NewTKAServer(
		api.WithAuthMiddleware(authMwMock),
		api.WithRetryAfterSeconds(10),
	)
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
