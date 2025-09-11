package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/models"
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
func (t *TKAServer) logout(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.logout")
	defer span.End()

	if signIn, err := t.auth.Status(ctx, userName); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting login status")
		writeHumaneError(ct, err, http.StatusNotFound)
		return
	} else {
		until := signIn.ValidUntil

		if !signIn.Provisioned {
			validity, err := time.ParseDuration(signIn.ValidityPeriod)
			if err != nil {
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		if err := t.auth.Logout(ctx, userName); err != nil {
			otelzap.L().WithError(err).ErrorContext(ctx, "Error logging out user")
			writeHumaneError(ct, err, http.StatusNotFound)
			return
		}

		ct.JSON(http.StatusOK, models.NewUserLoginResponse(signIn.Username, signIn.Role, until))
		return
	}
}
