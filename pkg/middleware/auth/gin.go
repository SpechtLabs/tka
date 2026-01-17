// Package auth provides authentication middleware for the TKA service.
// This package implements Tailscale-based authentication middleware that
// integrates with Gin HTTP framework to provide secure user authentication
// and capability-based authorization.
package auth

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mw "github.com/spechtlabs/tka/pkg/middleware"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/tshttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"tailscale.com/tailcfg"
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
type ginAuthMiddleware[capRule tshttp.TailscaleCapability] struct {
	capName     tailcfg.PeerCapability
	resolver    tshttp.WhoIsResolver
	allowTagged bool
	allowFunnel bool
}

// NewGinAuthMiddleware creates a new Tailscale authentication middleware for Gin.
// The middleware extracts capability rules from Tailscale ACL for each request,
// makes username and rule available via GetUsername() and GetCapability(),
// and rejects unauthorized users with appropriate HTTP status codes.
func NewGinAuthMiddleware[capRule tshttp.TailscaleCapability](resolver tshttp.WhoIsResolver, capName tailcfg.PeerCapability, opts ...Option[capRule]) mw.Middleware {
	mw := &ginAuthMiddleware[capRule]{
		capName:     capName,
		resolver:    resolver,
		allowTagged: false,
		allowFunnel: false,
	}

	for _, opt := range opts {
		opt(mw)
	}

	return mw
}

// Use installs the authentication middleware into the provided Gin engine.
// This method implements the middleware.Middleware interface.
// The middleware will be applied to all routes registered after this call.
// It performs authentication on every request and populates the Gin context
// with user information for downstream handlers.
func (m *ginAuthMiddleware[capRule]) Use(e *gin.Engine, tracer trace.Tracer) {
	e.Use(m.handler(tracer))
}

// UseGroup installs the authentication middleware into the provided Gin router group.
// This method implements the middleware.Middleware interface.
// The middleware will only be applied to routes within this specific group,
// allowing unauthenticated routes (like health checks) to exist alongside authenticated APIs.
func (m *ginAuthMiddleware[capRule]) UseGroup(rg *gin.RouterGroup, tracer trace.Tracer) {
	rg.Use(m.handler(tracer))
}

func (m *ginAuthMiddleware[capRule]) handler(tracer trace.Tracer) gin.HandlerFunc {
	return func(ct *gin.Context) {
		req := ct.Request

		ctx, span := tracer.Start(req.Context(), "Middleware.Auth")

		// Wide event context - these are captured by the defer closure below
		var (
			userName     string
			rejectReason string       //nolint:golint-sl // captured by defer, assigned in multiple branches
			statusCode   int          //nolint:golint-sl // captured by defer, assigned in multiple branches
			success      = true       //nolint:golint-sl // captured by defer, assigned in multiple branches
		)

		// Set span attributes for wide event data at end of auth
		defer func() {
			span.SetAttributes(
				attribute.String("auth.username", userName),
				attribute.String("auth.remote_addr", req.RemoteAddr),
				attribute.String("auth.path", req.URL.Path),
				attribute.Bool("auth.success", success),
			)

			if !success {
				span.SetAttributes(
					attribute.String("auth.reject_reason", rejectReason),
					attribute.Int("auth.status_code", statusCode),
				)
				span.SetStatus(codes.Error, rejectReason)
				otelzap.L().WarnContext(ctx, "auth rejected",
					zap.String("username", userName),
					zap.String("reject_reason", rejectReason),
					zap.Int("status_code", statusCode),
				)
			}

			span.End()
		}()

		// This URL is visited by the user who is being authenticated. If they are
		// visiting the URL over Funnel, that means they are not part of the
		// tailnet that they are trying to be authenticated for.
		if tshttp.IsFunnelRequest(ct.Request) && !m.allowFunnel {
			success, rejectReason, statusCode = false, "funnel_not_allowed", http.StatusForbidden
			ct.JSON(http.StatusForbidden, models.NewErrorResponse("Unauthorized request from Funnel"))
			ct.Abort()
			return
		}

		who, herr := m.resolver.WhoIs(ctx, req.RemoteAddr)
		if herr != nil {
			success, rejectReason, statusCode = false, "whois_failed", http.StatusInternalServerError
			ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error getting WhoIs", herr))
			ct.Abort()
			return
		}

		// Extract username from login name
		userName, _, _ = strings.Cut(who.LoginName, "@")

		if who.IsTagged() && !m.allowTagged {
			success, rejectReason, statusCode = false, "tagged_node_not_allowed", http.StatusBadRequest
			ct.JSON(http.StatusBadRequest, models.NewErrorResponse("tagged nodes not (yet) supported"))
			ct.Abort()
			return
		}

		rules, err := tailcfg.UnmarshalCapJSON[capRule](who.CapMap, m.capName)
		if err != nil {
			success, rejectReason, statusCode = false, "capability_unmarshal_failed", http.StatusBadRequest
			ct.JSON(http.StatusBadRequest, models.FromHumaneError(humane.Wrap(err, "Error unmarshaling api capability map", "Check the syntax of your api ACL for user "+userName+".")))
			ct.Abort()
			return
		}

		if len(rules) == 0 {
			success, rejectReason, statusCode = false, "no_capability_rules", http.StatusForbidden
			ct.JSON(http.StatusForbidden, models.NewErrorResponse("User not authorized"))
			ct.Abort()
			return
		}

		// If there are multiple rules, we need to sort them by priority.
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Priority() > rules[j].Priority()
		})

		// Check for duplicate priorities - accumulate instead of logging in loop
		var duplicatePriorities []int
		for i := 0; i < len(rules)-1; i++ {
			if rules[i].Priority() == rules[i+1].Priority() {
				duplicatePriorities = append(duplicatePriorities, rules[i].Priority())
			}
		}

		if len(duplicatePriorities) > 0 {
			err := humane.New("Multiple capability rules with the same priority found",
				"Please ensure that no two capability rules have the same priority as this will lead to an undefined behavior.",
			)
			success, rejectReason, statusCode = false, "duplicate_priority_rules", http.StatusBadRequest
			ct.JSON(http.StatusBadRequest, models.FromHumaneError(err))
			ct.Abort()
			return
		}

		SetUsername(ct, userName)
		SetCapability(ct, rules[0])

		ct.Next()
	}
}
