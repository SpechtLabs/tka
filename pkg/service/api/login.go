package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	globalModels "github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/spechtlabs/tka/pkg/service/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// login handles user authentication through Tailscale for the TKA service.
// @Summary       Authenticate user and provision Kubernetes credentials
// @Description   Authenticates a user through Tailscale, validates their capability rule, and provisions Kubernetes credentials
// @Tags          authentication
// @Accept        application/json
// @Produce       application/json
// @Success       202         {object}  models.UserLoginResponse  "Accepted - User authenticated and credentials are being provisioned"
// @Failure       400         {object}  models.ErrorResponse      "Bad Request - Tagged nodes not supported or error unmarshaling capability or multiple capability rules"
// @Failure       403         {object}  models.ErrorResponse      "Forbidden - Request from Funnel or no capability rule found"
// @Failure       422         {object}  models.ErrorResponse      "Unprocessable Entity - Invalid capability rule (period too short)"
// @Failure       500         {object}  models.ErrorResponse      "Internal Server Error - Error with WhoIs, parsing duration, or signing in user"
// @Router        /api/v1alpha1/login [post]
// @Security      TailscaleAuth
//
//nolint:golint-sl // Logs are in mutually exclusive error branches, only one executes per request
func (t *TKAServer) login(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)
	capRule := mwauth.GetCapability[capability.Rule](ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.login")
	defer span.End()

	// Set initial span attributes
	span.SetAttributes(attribute.String("login.username", userName))

	if capRule == nil {
		span.SetAttributes(
			attribute.String("login.status", "forbidden"),
			attribute.String("login.role", "unknown"),
		)
		span.SetStatus(codes.Error, "no capability rule found")
		otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.",
			zap.String("username", userName),
			zap.Int("http_status", http.StatusForbidden),
		)
		loginAttempts.WithLabelValues(userName, "unknown", "forbidden").Inc()
		ct.JSON(http.StatusForbidden, globalModels.NewErrorResponse("No grant found for user", nil))
		return
	}

	now := time.Now() //nolint:golint-sl // captures request timestamp for valid_until calculation
	role := capRule.Role
	span.SetAttributes(attribute.String("login.role", role))

	period, err := time.ParseDuration(capRule.Period)
	if err != nil {
		span.SetAttributes(attribute.String("login.status", "error"))
		span.SetStatus(codes.Error, "error parsing duration")
		span.RecordError(err)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusInternalServerError, globalModels.NewErrorResponse("Error parsing duration", err))
		return
	}

	span.SetAttributes(attribute.String("login.period", period.String()))

	if err := t.client.NewSignIn(ctx, userName, role, period); err != nil {
		span.SetAttributes(attribute.String("login.status", "error"))
		span.SetStatus(codes.Error, "error signing in user")
		span.RecordError(err)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
		loginAttempts.WithLabelValues(userName, role, "error").Inc()
		writeHumaneError(ct, err, http.StatusNotFound)
		return
	}

	// Set success attributes
	until := now.Add(period).Format(time.RFC3339)
	span.SetAttributes(
		attribute.String("login.status", "success"),
		attribute.String("login.valid_until", until),
	)

	// Track successful login metrics
	loginAttempts.WithLabelValues(userName, role, "success").Inc()

	ct.JSON(http.StatusAccepted, models.NewUserLoginResponse(userName, role, until))
}

// getLogin handles retrieving login status through Tailscale for the TKA service.
// @Summary       Get user authentication status
// @Description   Retrieves the current authentication status for a Tailscale user
// @Tags          authentication
// @Produce       application/json
// @Success       200         {object}  models.UserLoginResponse  "OK - Returns the current user authentication status"
// @Failure       400         {object}  models.ErrorResponse      "Bad Request - Tagged nodes not supported or error unmarshaling capability or multiple capability rules"
// @Failure       403         {object}  models.ErrorResponse      "Forbidden - Request from Funnel or no capability rule found"
// @Failure       500         {object}  models.ErrorResponse      "Internal Server Error - Error with WhoIs or retrieving user status"
// @Header        202         {integer} Retry-After               "Seconds until next poll recommended"
// @Router        /api/v1alpha1/login [get]
// @Security      TailscaleAuth
//
//nolint:golint-sl // Logs are in mutually exclusive error branches, only one executes per request
func (t *TKAServer) getLogin(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getLogin")
	defer span.End()

	// Set initial span attributes
	span.SetAttributes(attribute.String("get_login.username", userName))

	if signIn, err := t.client.GetStatus(ctx, userName); err != nil {
		span.SetAttributes(attribute.String("get_login.status", "error"))
		span.SetStatus(codes.Error, "error getting login status")
		span.RecordError(err)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting login status")
		// map k8s NotFound to 401 for this endpoint
		writeHumaneError(ct, err, http.StatusUnauthorized)
		return
	} else {
		status := http.StatusOK
		until := signIn.ValidUntil //nolint:golint-sl // may be updated in provisioning check below

		span.SetAttributes(
			attribute.String("get_login.role", signIn.Role),
			attribute.Bool("get_login.provisioned", signIn.Provisioned),
		)

		if !signIn.Provisioned {
			status = http.StatusAccepted
			ct.Header("Retry-After", strconv.Itoa(t.retryAfterSeconds))

			validity, err := time.ParseDuration(signIn.ValidityPeriod)
			if err != nil {
				span.SetAttributes(attribute.String("get_login.status", "error"))
				span.SetStatus(codes.Error, "error parsing duration")
				span.RecordError(err)
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, globalModels.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		span.SetAttributes(
			attribute.String("get_login.status", "success"),
			attribute.String("get_login.valid_until", until),
			attribute.Int("get_login.http_status", status),
		)

		ct.JSON(status, models.NewUserLoginResponse(signIn.Username, signIn.Role, until))
		return
	}
}
