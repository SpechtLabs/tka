// Package auth provides authentication middleware for the TKA service.
// This package implements Tailscale-based authentication middleware that
// integrates with Gin HTTP framework to provide secure user authentication
// and capability-based authorization.
package auth

import (
	"net/http"
	"strings"

	// misc
	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"tailscale.com/tailcfg"

	// o11y
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.opentelemetry.io/otel/trace"

	// tka
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/tailscale"
)

// ginAuthMiddleware provides Tailscale-based authentication middleware for Gin HTTP servers.
// It implements the middleware.Middleware interface and uses generic types to support
// different capability rule structures.
//
// The middleware performs these authentication steps:
//  1. Rejects requests from Tailscale Funnel (external access)
//  2. Performs WhoIs lookup on the client's IP address
//  3. Rejects tagged nodes (service accounts)
//  4. Extracts and validates capability rules from Tailscale ACLs
//  5. Stores username and capability in Gin context for handlers
type ginAuthMiddleware[capRule any] struct {
	capName  tailcfg.PeerCapability
	resolver tailscale.WhoIsResolver
}

// NewGinAuthMiddleware creates a new Tailscale authentication middleware for Gin.
// The middleware extracts capability rules from Tailscale ACL for each request,
// makes username and rule available via GetUsername() and GetCapability(),
// and rejects unauthorized users with appropriate HTTP status codes.
func NewGinAuthMiddleware[capRule any](resolver tailscale.WhoIsResolver, capName tailcfg.PeerCapability) *ginAuthMiddleware[capRule] {
	return &ginAuthMiddleware[capRule]{
		capName:  capName,
		resolver: resolver,
	}
}

// Use installs the authentication middleware into the provided Gin engine.
// This method implements the middleware.Middleware interface.
// The middleware will be applied to all routes registered after this call.
// It performs authentication on every request and populates the Gin context
// with user information for downstream handlers.
func (m *ginAuthMiddleware[capRule]) Use(e *gin.Engine, tracer trace.Tracer) {
	e.Use(m.handler(tracer))
}

func (m *ginAuthMiddleware[capRule]) handler(tracer trace.Tracer) gin.HandlerFunc {
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

		who, herr := m.resolver.WhoIs(ctx, req.RemoteAddr)
		if herr != nil {
			otelzap.L().WithError(herr).ErrorContext(ctx, "Error getting WhoIs")
			ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error getting WhoIs", herr))
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

		SetUsername(ct, userName)
		SetCapability(ct, rules[0])

		ct.Next()
	}
}
