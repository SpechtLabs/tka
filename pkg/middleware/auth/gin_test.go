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
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/capability"
	ts "github.com/spechtlabs/tka/pkg/tshttp"
	"github.com/spechtlabs/tka/pkg/tshttp/mock"
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
		c.JSON(http.StatusOK, gin.H{
			"user": mwauth.GetUsername(c),
			"role": mwauth.GetCapability[capability.Rule](c).Role},
		)
	})

	// Return the router and a recorder
	return r, httptest.NewRecorder()
}

type whoisResponse struct {
	ts.WhoIsInfo
	whoisErr error
}

func TestGinAuthMiddleware(t *testing.T) {
	capName := tailcfg.PeerCapability("specht-labs.de/cap/tka")

	viewer := capability.Rule{Role: "viewer", Period: "10m", RulePriority: 100}
	admin := capability.Rule{Role: "admin", Period: "10m", RulePriority: 200}
	admin2 := capability.Rule{Role: "admin", Period: "10m", RulePriority: 100}

	viewerB, _ := json.Marshal(viewer)
	adminB, _ := json.Marshal(admin)
	admin2B, _ := json.Marshal(admin2)

	cases := []struct {
		name          string
		whoisResponse whoisResponse
		headers       map[string]string
		allowTagged   bool
		allowFunnel   bool
		wantStatus    int
		wantUser      string
		wantRole      string
		wantError     string
	}{
		{
			name: "success single rule extracts user and passes",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    buildCap(t, capName, viewer),
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusOK,
			wantUser:    "alice",
			wantRole:    "viewer",
		},
		{
			name: "funnel request is forbidden",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    buildCap(t, capName, viewer),
				},
			},
			headers:     map[string]string{"Tailscale-Funnel-Request": "1"},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusForbidden,
			wantError:   "Unauthorized request from Funnel",
		},
		{
			name: "funnel request is allowed, only if allowed",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    buildCap(t, capName, viewer),
				},
			},
			headers:     map[string]string{"Tailscale-Funnel-Request": "1"},
			allowFunnel: true,
			allowTagged: false,
			wantStatus:  http.StatusOK,
			wantRole:    "viewer",
		},
		{
			name: "whois error yields 500",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    buildCap(t, capName, viewer),
				},
				whoisErr: context.DeadlineExceeded,
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "Error getting WhoIs",
		},
		{
			name: "tagged nodes are rejected",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{"tag:test"},
					CapMap:    buildCap(t, capName, viewer),
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusBadRequest,
			wantError:   "tagged nodes not (yet) supported",
		},
		{
			name: "tagged nodes are allowed, only if allowed",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{"tag:test"},
					CapMap:    buildCap(t, capName, viewer),
				},
			},
			allowTagged: true,
			allowFunnel: false,
			wantStatus:  http.StatusOK,
			wantRole:    "viewer",
		},
		{
			name: "no rule found -> 403",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    tailcfg.PeerCapMap{},
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusForbidden,
			wantError:   "User not authorized",
		},
		{
			name: "malformed rule -> 400",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage("not-json")}},
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusBadRequest,
			wantError:   "Error unmarshaling api capability map",
		},
		{
			name: "multiple rules -> use highest priority rule",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage(viewerB), tailcfg.RawMessage(adminB)}},
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusOK,
			wantUser:    "alice",
			wantRole:    "admin",
		},
		{
			name: "multiple rules with same priority -> 400",
			whoisResponse: whoisResponse{
				WhoIsInfo: ts.WhoIsInfo{
					LoginName: "alice@example.com",
					Tags:      []string{},
					CapMap:    tailcfg.PeerCapMap{capName: []tailcfg.RawMessage{tailcfg.RawMessage(viewerB), tailcfg.RawMessage(admin2B)}},
				},
			},
			allowFunnel: false,
			allowTagged: false,
			wantStatus:  http.StatusBadRequest,
			wantError:   "Multiple capability rules with the same priority found",
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
				mock.WithWhoIsResponse(req.RemoteAddr, &tc.whoisResponse.WhoIsInfo),
				mock.WithWhoIsError(req.RemoteAddr, tc.whoisResponse.whoisErr),
			)

			// 3. Setup auth middleware using our mock whois resolver
			authMiddleware := mwauth.NewGinAuthMiddleware(whoIsResolver, capName,
				mwauth.AllowFunnelRequest[capability.Rule](tc.allowFunnel),
				mwauth.AllowTaggedNodes[capability.Rule](tc.allowTagged),
			)

			// 4. Setup router using our auth middleware
			r, w := setupRouter(t, authMiddleware)

			// 5. Serve request
			r.ServeHTTP(w, req)

			// 6. Profit!
			require.Equal(t, tc.wantStatus, w.Code)

			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

				if tc.wantUser != "" {
					require.Contains(t, resp, "user")
					require.Equal(t, tc.wantUser, resp["user"])
				}

				if tc.wantRole != "" {
					require.Contains(t, resp, "role")
					require.Equal(t, tc.wantRole, resp["role"])
				}
			} else {
				var err models.ErrorResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &err))
				require.Equal(t, tc.wantError, err.Message)
			}
		})
	}
}
