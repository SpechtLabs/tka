package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	globalModels "github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/auth/models"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"go.uber.org/zap"
)

// login handles user authentication through Tailscale for the TKA service
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
func (t *TKAServer) login(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)
	capRule := mwauth.GetCapability[capability.Rule](ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.login")
	defer span.End()

	if capRule == nil {
		otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.")
		loginAttempts.WithLabelValues(userName, "unknown", "forbidden").Inc()
		ct.JSON(http.StatusForbidden, globalModels.NewErrorResponse("No grant found for user", nil))
		return
	}

	now := time.Now()
	role := capRule.Role
	period, err := time.ParseDuration(capRule.Period)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusInternalServerError, globalModels.NewErrorResponse("Error parsing duration", err))
		return
	}

	if err := t.client.NewSignIn(ctx, userName, role, period); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
		loginAttempts.WithLabelValues(userName, role, "error").Inc()
		writeHumaneError(ct, err, http.StatusNotFound)
		return
	}

	otelzap.L().InfoContext(ctx,
		"User login request was successful and is now awaiting the provisioning of the Kubernetes credentials",
		zap.String("user", userName),
		zap.String("role", role),
		zap.String("now", now.Format(time.RFC3339)),
		zap.String("period", period.String()),
		zap.String("until", now.Add(period).Format(time.RFC3339)),
	)

	// Track successful login metrics
	loginAttempts.WithLabelValues(userName, role, "success").Inc()

	ct.JSON(http.StatusAccepted, models.NewUserLoginResponse(userName, role, now.Add(period).Format(time.RFC3339)))
}

// getLogin handles retrieving login status through Tailscale for the TKA service
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
func (t *TKAServer) getLogin(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getLogin")
	defer span.End()

	if signIn, err := t.client.GetStatus(ctx, userName); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting login status")
		// map k8s NotFound to 401 for this endpoint
		writeHumaneError(ct, err, http.StatusUnauthorized)
		return
	} else {
		status := http.StatusOK
		until := signIn.ValidUntil

		if !signIn.Provisioned {
			status = http.StatusAccepted
			ct.Header("Retry-After", strconv.Itoa(t.retryAfterSeconds))

			validity, err := time.ParseDuration(signIn.ValidityPeriod)
			if err != nil {
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, globalModels.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		ct.JSON(status, models.NewUserLoginResponse(signIn.Username, signIn.Role, until))
		return
	}
}
