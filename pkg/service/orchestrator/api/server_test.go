package api_test

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	mwMock "github.com/spechtlabs/tka/pkg/middleware/auth/mock"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/spechtlabs/tka/pkg/service/orchestrator/api"
	"github.com/stretchr/testify/require"
)

func TestNewTKAServer_OrchestratorRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authMwMock := &mwMock.AuthMiddleware{Username: "alice", Rule: capability.Rule{}, OmitRule: false}

	s, err := api.NewTKAServer(nil,
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
		http.MethodPost + " /api/v1alpha1/login":                                  {Expected: false, Seen: false},
		http.MethodGet + " /api/v1alpha1/login":                                   {Expected: false, Seen: false},
		http.MethodGet + " /api/v1alpha1/kubeconfig":                              {Expected: false, Seen: false},
		http.MethodPost + " /api/v1alpha1/logout":                                 {Expected: false, Seen: false},
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
