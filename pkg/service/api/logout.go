package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	globalModels "github.com/spechtlabs/tka/pkg/models"
	"github.com/spechtlabs/tka/pkg/service/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// logout handles user logout from TKA service
// @Summary       Log out authenticated user
// @Description   Revokes Kubernetes credentials for the authenticated Tailscale user
// @Tags          authentication
// @Produce       application/json
// @Success       200         {object}  models.UserLoginResponse       "OK - User successfully logged out with login info"
// @Failure       400         {object}  models.ErrorResponse           "Bad Request - Tagged nodes not supported or error unmarshaling capability or multiple capability rules"
// @Failure       403         {object}  models.ErrorResponse           "Forbidden - Request from Funnel or no capability rule found"
// @Failure       404         {object}  models.ErrorResponse           "Not Found - User not authenticated"
// @Failure       500         {object}  models.ErrorResponse           "Internal Server Error - Error with WhoIs, parsing duration, or during logout process"
// @Router        /api/v1alpha1/logout [post]
// @Security      TailscaleAuth
//
//nolint:golint-sl // Logs are in mutually exclusive error branches, only one executes per request
func (t *TKAServer) logout(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.logout")
	defer span.End()

	// Set initial span attributes
	span.SetAttributes(attribute.String("logout.username", userName))

	if signIn, err := t.client.GetStatus(ctx, userName); err != nil {
		span.SetAttributes(attribute.String("logout.status", "error_get_status"))
		span.SetStatus(codes.Error, "error getting login status")
		span.RecordError(err)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting login status")
		writeHumaneError(ct, err, http.StatusNotFound)
		return
	} else {
		span.SetAttributes(
			attribute.String("logout.role", signIn.Role),
			attribute.Bool("logout.was_provisioned", signIn.Provisioned),
		)

		until := signIn.ValidUntil

		if !signIn.Provisioned {
			validity, err := time.ParseDuration(signIn.ValidityPeriod)
			if err != nil {
				span.SetAttributes(attribute.String("logout.status", "error_parse_duration"))
				span.SetStatus(codes.Error, "error parsing duration")
				span.RecordError(err)
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, globalModels.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		if err := t.client.DeleteSignIn(ctx, userName); err != nil {
			span.SetAttributes(attribute.String("logout.status", "error_delete"))
			span.SetStatus(codes.Error, "error logging out user")
			span.RecordError(err)
			otelzap.L().WithError(err).ErrorContext(ctx, "Error logging out user")
			writeHumaneError(ct, err, http.StatusNotFound)
			return
		}

		span.SetAttributes(
			attribute.String("logout.status", "success"),
			attribute.Int("logout.http_status", http.StatusOK),
		)

		ct.JSON(http.StatusOK, models.NewUserLoginResponse(signIn.Username, signIn.Role, until))
		return
	}
}
