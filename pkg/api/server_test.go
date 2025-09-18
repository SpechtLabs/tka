package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/api"
	mwMock "github.com/spechtlabs/tka/pkg/middleware/auth/mock"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/spechtlabs/tka/pkg/service/mock"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newTestServer(t *testing.T, auth service.Service, rule capability.Rule) (*api.TKAServer, *httptest.Server) {
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
	s, err := api.NewTKAServer(nil, nil, api.WithAuthMiddleware(authMwMock))
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NotNil(t, s.Engine())

	authSvc := mock.NewMockAuthService()

	require.Error(t, s.LoadApiRoutes(nil))
	require.NoError(t, s.LoadApiRoutes(authSvc))

	// Maps the key to if the route is expected
	expected := map[string]struct {
		Expected bool
		Seen     bool
	}{
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:          {Expected: true, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:           {Expected: true, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.KubeconfigApiRoute:      {Expected: true, Seen: false},
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LogoutApiRoute:         {Expected: true, Seen: false},
		http.MethodGet + " " + api.OrchestratorRouteV1Alpha1 + api.ClustersRoute:  {Expected: false, Seen: false},
		http.MethodPost + " " + api.OrchestratorRouteV1Alpha1 + api.ClustersRoute: {Expected: false, Seen: false},
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

func TestNewTKAServer_OrchestratorRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}

	s, err := api.NewTKAServer(nil, nil,
		api.WithAuthMiddleware(authMwMock),
	)

	require.NoError(t, err)
	require.NotNil(t, s)
	require.NotNil(t, s.Engine())

	require.NoError(t, s.LoadOrchestratorRoutes())

	// Maps the key to if the route is expected
	expected := map[string]struct {
		Expected bool
		Seen     bool
	}{
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:          {Expected: false, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:           {Expected: false, Seen: false},
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.KubeconfigApiRoute:      {Expected: false, Seen: false},
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LogoutApiRoute:         {Expected: false, Seen: false},
		http.MethodGet + " " + api.OrchestratorRouteV1Alpha1 + api.ClustersRoute:  {Expected: true, Seen: false},
		http.MethodPost + " " + api.OrchestratorRouteV1Alpha1 + api.ClustersRoute: {Expected: true, Seen: false},
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
