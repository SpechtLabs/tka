package tailscale

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/tailscale"
	"go.opentelemetry.io/otel/trace"
	"tailscale.com/tailcfg"
)

// GinAuthMiddleware provides a tailscale-backed auth middleware implementing middleware/auth.Middleware.
type GinAuthMiddleware[capRule any] struct {
	whoIs   tailscale.WhoIsFunc
	capName tailcfg.PeerCapability
	server  *tailscale.Server
}

// NewGinAuthMiddlewareFromServer keeps backward compatibility by deriving the resolver from *tailscale.Server.
func NewGinAuthMiddlewareFromServer[capRule any](tsServer *tailscale.Server, capName tailcfg.PeerCapability) *GinAuthMiddleware[capRule] {
	return &GinAuthMiddleware[capRule]{
		whoIs:   tsServer.Identity(),
		capName: capName,
		server:  tsServer,
	}
}

// NewGinAuthMiddleware constructs middleware from a WhoIsFunc, enabling unit tests and alternative identity sources.
func NewGinAuthMiddleware[capRule any](who tailscale.WhoIsFunc, capName tailcfg.PeerCapability) *GinAuthMiddleware[capRule] {
	return &GinAuthMiddleware[capRule]{
		whoIs:   who,
		capName: capName,
	}
}

func (m *GinAuthMiddleware[capRule]) Use(e *gin.Engine, tracer trace.Tracer) {
	e.Use(m.handler(tracer))
}

func (m *GinAuthMiddleware[capRule]) handler(tracer trace.Tracer) gin.HandlerFunc {
	return func(ct *gin.Context) {
		req := ct.Request

		ctx, span := tracer.Start(req.Context(), "Middleware.Auth")
		defer span.End()

		// This URL is visited by the user who is being authenticated. If they are
		// visiting the URL over Funnel, that means they are not part of the
		// tailnet that they are trying to be authenticated for.
		if tailscale.IsFunnelRequest(ct.Request) {
			otelzap.L().ErrorContext(ctx, "Unauthorized request from Funnel")
			ct.JSON(http.StatusForbidden, models.NewErrorResponse("Unauthorized request from Funnel", nil))
			ct.Abort()
			return
		}

		resolver := m.whoIs
		if resolver == nil && m.server != nil {
			resolver = m.server.Identity()
		}
		if resolver == nil {
			otelzap.L().ErrorContext(ctx, "Tailscale identity not initialized")
			ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Tailscale identity not initialized", nil))
			ct.Abort()
			return
		}

		who, err := resolver(ctx, req.RemoteAddr)
		if err != nil {
			otelzap.L().WithError(err).ErrorContext(ctx, "Error getting WhoIs")
			ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error getting WhoIs", err))
			ct.Abort()
			return
		}

		// not sure if this is the right thing to do...
		userName, _, _ := strings.Cut(who.LoginName, "@")
		if who.IsTagged {
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

		auth.SetUsername(ct, userName)
		auth.SetCapability(ct, rules[0])

		ct.Next()
	}
}
