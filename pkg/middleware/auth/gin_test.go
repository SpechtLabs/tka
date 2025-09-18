package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	mw "github.com/spechtlabs/tka/pkg/middleware"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	ts "github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spechtlabs/tka/pkg/tailscale/mock"
	"github.com/stretchr/testify/require"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"tailscale.com/tailcfg"
)

// helper to build a cap map with one rule for a capability name.
func buildCap(t *testing.T, capName tailcfg.PeerCapability, rule any) tailcfg.PeerCapMap {
	t.Helper()

	b, _ := json.Marshal(rule)
	vals := []tailcfg.RawMessage{tailcfg.RawMessage(b)}
	return tailcfg.PeerCapMap{capName: vals}
}

// common setup for middleware test: returns router and recorder
func setupRouter(t *testing.T, mw mw.Middleware) (*gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// use a noop tracer provider
	tr := nooptrace.NewTracerProvider().Tracer("test")

	// Load the auth middleware
	mw.Use(r, tr)

	// Add a test route that returns the username from the context
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": mwauth.GetUsername(c)})
	})

	// Return the router and a recorder
	return r, httptest.NewRecorder()
}

func TestGinAuthMiddleware(t *testing.T) {
	capName := tailcfg.PeerCapability("specht-labs.de/cap/tka")

	b1, _ := json.Marshal(map[string]string{"role": "a"})
	b2, _ := json.Marshal(map[string]string{"role": "b"})

	cases := []struct {
		name         string
		capMap       tailcfg.PeerCapMap
		err          error
		tagged       bool
		headers      map[string]string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "success single rule extracts user and passes",
			capMap:       buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}),
			err:          nil,
			wantStatus:   http.StatusOK,
			wantContains: "alice",
		},
		{
			name:       "funnel request is forbidden",
			capMap:     buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}),
			err:        nil,
			headers:    map[string]string{"Tailscale-Funnel-Request": "1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "whois error yields 500",
			capMap:     nil,
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "tagged nodes are rejected",
			capMap:     buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}),
			err:        nil,
			tagged:     true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no rule found -> 403",
			capMap:     tailcfg.PeerCapMap{},
			err:        nil,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "malformed rule -> 400",
			capMap:     tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage("not-json")}},
			err:        nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "multiple rules -> 400",
			capMap:     tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage(b1), tailcfg.RawMessage(b2)}},
			err:        nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Setup request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			// 2. Setup whois resolver with source from req
			whoIsResolver := mock.NewMockWhoIsResolver(
				mock.WithWhoIsResponse(req.RemoteAddr,
					&ts.WhoIsInfo{
						LoginName: "alice@example.com",
						IsTagged:  tc.tagged,
						CapMap:    tc.capMap,
					},
				),
				mock.WithWhoIsError(req.RemoteAddr, tc.err),
			)

			// 3. Setup auth middleware using our mock whois resolver
			authMiddleware := mwauth.NewGinAuthMiddleware[map[string]string](whoIsResolver, capName)

			// 4. Setup router using our auth middleware
			r, w := setupRouter(t, authMiddleware)

			// 5. Serve request
			r.ServeHTTP(w, req)

			// 6. Profit!
			require.Equal(t, tc.wantStatus, w.Code)
			if tc.wantContains != "" {
				require.Contains(t, w.Body.String(), tc.wantContains)
			}
		})
	}
}
