package tailscale

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/models"
	"go.opentelemetry.io/otel/trace"
	"tailscale.com/tailcfg"
)

// Context keys for Tailscale authentication data
const (
	// ContextKeyTailscaleUser is the key to retrieve the username from the context
	ContextKeyTailscaleUser = "tailscale_username"

	// ContextKeyTailscaleCapRule is the key to retrieve the capability rule from the context
	ContextKeyTailscaleCapRule = "tailscale_cap_rule"
)

// GetTailscaleUsername retrieves the Tailscale username from the context
func GetTailscaleUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextKeyTailscaleUser); exists {
		return username.(string)
	}
	return ""
}

// GetTailscaleCapRule retrieves the Tailscale capability rule from the context
func GetTailscaleCapRule[capRule any](c *gin.Context) *capRule {
	if rule, exists := c.Get(ContextKeyTailscaleCapRule); exists {
		if r, ok := rule.(capRule); ok {
			return &r
		}
		// Handle case where it's already a pointer
		if r, ok := rule.(*capRule); ok {
			return r
		}
	}

	// Return nil to indicate no rule was found
	return nil
}

type GinAuthMiddleware[capRule any] struct {
	tsServer *Server
	capName  tailcfg.PeerCapability
}

func NewGinAuthMiddleware[capRule any](tsServer *Server, capName tailcfg.PeerCapability) *GinAuthMiddleware[capRule] {
	return &GinAuthMiddleware[capRule]{
		tsServer: tsServer,
		capName:  capName,
	}
}

func (m *GinAuthMiddleware[capRule]) Use(e *gin.Engine, tracer trace.Tracer) {
	e.Use(m.TailscaleAuthHandlerFunc(tracer))
}

func (m *GinAuthMiddleware[capRule]) TailscaleAuthHandlerFunc(tracer trace.Tracer) gin.HandlerFunc {
	return func(ct *gin.Context) {
		req := ct.Request

		ctx, span := tracer.Start(req.Context(), "TKAServer.login")
		defer span.End()

		// This URL is visited by the user who is being authenticated. If they are
		// visiting the URL over Funnel, that means they are not part of the
		// tailnet that they are trying to be authenticated for.
		if IsFunnelRequest(ct.Request) {
			otelzap.L().ErrorContext(ctx, "Unauthorized request from Funnel")
			ct.JSON(http.StatusForbidden, models.NewErrorResponse("Unauthorized request from Funnel", nil))
			ct.Abort()
			return
		}

		who, err := m.tsServer.LC().WhoIs(ctx, req.RemoteAddr)
		if err != nil {
			otelzap.L().WithError(err).ErrorContext(ctx, "Error getting WhoIs")
			ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error getting WhoIs", err))
			ct.Abort()
			return
		}

		// not sure if this is the right thing to do...
		userName, _, _ := strings.Cut(who.UserProfile.LoginName, "@")
		n := who.Node.View()
		if n.IsTagged() {
			otelzap.L().ErrorContext(ctx, "tagged nodes not (yet) supported")
			ct.JSON(http.StatusBadRequest, models.NewErrorResponse("tagged nodes not (yet) supported", nil))
			ct.Abort()
			return
		}

		rules, err := tailcfg.UnmarshalCapJSON[capRule](who.CapMap, m.capName)
		if err != nil {
			otelzap.L().WithError(err).ErrorContext(ctx, "Error unmarshaling capability")
			ct.JSON(http.StatusBadRequest, models.FromHumaneError(humane.Wrap(err, "Error unmarshaling api capability map", "Check the syntax of your api ACL for user "+userName+".")))
			ct.Abort()
			return
		}

		if len(rules) == 0 {
			otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.")
			ct.JSON(http.StatusForbidden, models.NewErrorResponse("User not authorized", nil))
			ct.Abort()
			return
		}

		if len(rules) > 1 {
			// TODO(cedi): unsure what to do when having more than one cap...
			otelzap.L().ErrorContext(ctx, "More than one capability rule found")
			ct.JSON(http.StatusBadRequest, models.FromHumaneError(humane.New("More than one capability rule found", "Please ensure that you only have one capability rule for your user.", "If you have more than one, please contact the administrator of this system.")))
			ct.Abort()
			return
		}

		// Store authentication data in the context for use in handlers
		ct.Set(ContextKeyTailscaleUser, userName)
		ct.Set(ContextKeyTailscaleCapRule, rules[0])

		ct.Next()
	}
}
