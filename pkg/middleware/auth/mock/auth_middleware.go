package mock

import (
	"github.com/gin-gonic/gin"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"go.opentelemetry.io/otel/trace"
)

// AuthMiddleware is a mock implementation of middleware.Middleware for testing.
// It bypasses real Tailscale authentication and injects fixed user credentials
// into the Gin context, allowing tests to simulate different user scenarios.
type AuthMiddleware struct {
	// Username is the fixed username to inject into all requests
	Username string
	// Rule is the fixed capability rule to inject into all requests
	Rule capability.Rule
	// OmitRule skips setting the capability rule (simulates unauthorized users)
	OmitRule bool
}

// Use installs the mock authentication middleware into the provided Gin engine.
// This method implements the middleware.Middleware interface for testing.
//
// The mock middleware simulates successful authentication for testing scenarios
// where predictable user context is needed without network dependencies.
func (m *AuthMiddleware) Use(e *gin.Engine, _ trace.Tracer) {
	e.Use(func(c *gin.Context) {
		mwauth.SetUsername(c, m.Username)
		if !m.OmitRule {
			mwauth.SetCapability(c, m.Rule)
		}
	})
}

// UseGroup installs the mock authentication middleware into the provided Gin router group.
// This method implements the middleware.Middleware interface for testing.
//
// The mock middleware simulates successful authentication for testing scenarios
// where predictable user context is needed without network dependencies.
func (m *AuthMiddleware) UseGroup(rg *gin.RouterGroup, _ trace.Tracer) {
	rg.Use(func(c *gin.Context) {
		mwauth.SetUsername(c, m.Username)
		if !m.OmitRule {
			mwauth.SetCapability(c, m.Rule)
		}
	})
}
