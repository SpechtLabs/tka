package tailscale_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/stretchr/testify/require"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
	"tailscale.com/tailcfg"
)

// mockWhoIs constructs a WhoIsFunc returning the provided values.
func mockWhoIs(t *testing.T, isTagged bool, cap tailcfg.PeerCapMap, err error) tailscale.WhoIsFunc {
	t.Helper()

	return func(ctx context.Context, remoteAddr string) (*tailscale.WhoIsInfo, error) {
		if err != nil {
			return nil, err
		}
		return &tailscale.WhoIsInfo{LoginName: "alice@example.com", IsTagged: isTagged, CapMap: cap}, nil
	}
}

// helper to build a cap map with one rule for a capability name.
func buildCap(t *testing.T, capName tailcfg.PeerCapability, rule any) tailcfg.PeerCapMap {
	t.Helper()

	b, _ := json.Marshal(rule)
	vals := []tailcfg.RawMessage{tailcfg.RawMessage(b)}
	return tailcfg.PeerCapMap{capName: vals}
}

// common setup for middleware test: returns router and recorder
func setupRouter(t *testing.T, mw *tailscale.GinAuthMiddleware[map[string]string]) (*gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// use a noop tracer provider
	tr := nooptrace.NewTracerProvider().Tracer("test")
	mw.Use(r, tr)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user": tailscale.GetTailscaleUsername(c)})
	})
	return r, httptest.NewRecorder()
}

func TestGinAuthMiddleware(t *testing.T) {
	capName := tailcfg.PeerCapability("specht-labs.de/cap/tka")

	b1, _ := json.Marshal(map[string]string{"role": "a"})
	b2, _ := json.Marshal(map[string]string{"role": "b"})

	cases := []struct {
		name         string
		who          tailscale.WhoIsFunc
		headers      map[string]string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "success single rule extracts user and passes",
			who:          mockWhoIs(t, false, buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}), nil),
			wantStatus:   http.StatusOK,
			wantContains: "alice",
		},
		{
			name:       "funnel request is forbidden",
			who:        mockWhoIs(t, false, buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}), nil),
			headers:    map[string]string{"Tailscale-Funnel-Request": "1"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "whois error yields 500",
			who:        mockWhoIs(t, false, nil, context.DeadlineExceeded),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "tagged nodes are rejected",
			who:        mockWhoIs(t, true, buildCap(t, capName, map[string]string{"role": "viewer", "period": "10m"}), nil),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no rule found -> 403",
			who:        mockWhoIs(t, false, tailcfg.PeerCapMap{}, nil),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "malformed rule -> 400",
			who:        mockWhoIs(t, false, tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage("not-json")}}, nil),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "multiple rules -> 400",
			who:        mockWhoIs(t, false, tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage(b1), tailcfg.RawMessage(b2)}}, nil),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mw := tailscale.NewGinAuthMiddleware[map[string]string](tc.who, capName)
			r, w := setupRouter(t, mw)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			if tc.wantContains != "" {
				require.Contains(t, w.Body.String(), tc.wantContains)
			}
		})
	}
}
