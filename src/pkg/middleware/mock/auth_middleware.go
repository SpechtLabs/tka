package mock

import (
	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tka/pkg/auth/capability"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"go.opentelemetry.io/otel/trace"
)

// AuthMiddleware is a test helper that injects a fixed user and capability rule.
type AuthMiddleware struct {
	Username string
	Rule     capability.Rule
	OmitRule bool
}

func (m *AuthMiddleware) Use(e *gin.Engine, _ trace.Tracer) {
	e.Use(func(c *gin.Context) {
		mwauth.SetUsername(c, m.Username)
		if !m.OmitRule {
			mwauth.SetCapability(c, m.Rule)
		}
	})
}
