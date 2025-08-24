package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/models"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
// @Security      tailscaleAuth
func (t *TKAServer) login(ct *gin.Context) {
	req := ct.Request
	userName := tailscale.GetTailscaleUsername(ct)
	capRule := tailscale.GetTailscaleCapRule[capRule](ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.login")
	defer span.End()

	if capRule == nil {
		otelzap.L().ErrorContext(ctx, "No capability rule found for user. Assuming unauthorized.")
		ct.JSON(http.StatusForbidden, models.NewErrorResponse("No grant found for user", nil))
		return
	}

	now := time.Now()
	role := capRule.Role
	period, err := time.ParseDuration(capRule.Period)
	if err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
		ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error parsing duration", err))
		return
	}

	if period < operator.MinSigninValidity {
		err := humane.New("`period` may not specify a duration less than 10 minutes",
			fmt.Sprintf("Specify a period greater than 10 minutes in your api ACL for user %s", userName),
		)
		otelzap.L().WithError(err).ErrorContext(ctx, "Invalid capRule")
		ct.JSON(http.StatusUnprocessableEntity, models.NewErrorResponse("Invalid capRule", err))
		return
	}

	if err := t.operator.SignInUser(ctx, userName, role, period); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
		ct.JSON(http.StatusInternalServerError, models.FromHumaneError(err))
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
// @Router        /api/v1alpha1/login [get]
// @Security      TailscaleAuth
func (t *TKAServer) getLogin(ct *gin.Context) {
	req := ct.Request
	userName := tailscale.GetTailscaleUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getLogin")
	defer span.End()

	if signIn, err := t.operator.GetSignInUser(ctx, userName); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")
		if err.Cause() != nil && k8serrors.IsNotFound(err.Cause()) {
			ct.JSON(http.StatusUnauthorized, models.FromHumaneError(err))
			return
		} else {
			ct.JSON(http.StatusInternalServerError, models.FromHumaneError(err))
			return
		}
	} else {
		status := http.StatusOK
		until := signIn.Status.ValidUntil

		if !signIn.Status.Provisioned {
			status = http.StatusAccepted
			ct.Header("Retry-After", strconv.Itoa(t.retryAfterSeconds))

			validity, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
			if err != nil {
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		ct.JSON(status, models.NewUserLoginResponse(signIn.Spec.Username, signIn.Spec.Role, until))
		return
	}
}
