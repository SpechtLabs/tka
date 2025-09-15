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
	"github.com/spechtlabs/tka/pkg/auth"
	"github.com/spechtlabs/tka/pkg/auth/capability"
	mwMock "github.com/spechtlabs/tka/pkg/middleware/mock"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newTestServer(t *testing.T, auth auth.Service, rule capability.Rule) (*api.TKAServer, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	srv, err := api.NewTKAServer(
		nil,
		nil,
		api.WithAuthService(auth),
		api.WithAuthMiddleware(&mwMock.AuthMiddleware{Username: "alice", Rule: rule, OmitRule: rule.Role == "" && rule.Period == ""}),
	)
	require.NoError(t, err)

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
	s, err := api.NewTKAServer(nil, nil)
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NotNil(t, s.Engine())

	s.LoadApiRoutes()

	expected := map[string]bool{
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:     false,
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.LoginApiRoute:      false,
		http.MethodGet + " " + api.ApiRouteV1Alpha1 + api.KubeconfigApiRoute: false,
		http.MethodPost + " " + api.ApiRouteV1Alpha1 + api.LogoutApiRoute:    false,
	}
	for _, r := range s.Engine().Routes() {
		key := r.Method + " " + r.Path
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}
	for k, seen := range expected {
		require.True(t, seen, "missing route %s", k)
	}
}
