package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/models"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	userName := tailscale.GetTailscaleUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getLogin")
	defer span.End()

	if signIn, err := t.operator.GetSignInUser(ctx, userName); err != nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")
		if err.Cause() != nil && k8serrors.IsNotFound(err.Cause()) {
			ct.JSON(http.StatusNotFound, models.FromHumaneError(err))
			return
		} else {
			ct.JSON(http.StatusInternalServerError, models.FromHumaneError(err))
			return
		}
	} else {
		until := signIn.Status.ValidUntil

		if !signIn.Status.Provisioned {
			validity, err := time.ParseDuration(signIn.Spec.ValidityPeriod)
			if err == nil {
				otelzap.L().WithError(err).ErrorContext(ctx, "Error parsing duration")
				ct.JSON(http.StatusInternalServerError, models.NewErrorResponse("Error parsing duration", err))
				return
			}
			until = time.Now().Add(validity).Format(time.RFC3339)
		}

		if err := t.operator.LogOutUser(ctx, userName); err != nil {
			otelzap.L().WithError(err).ErrorContext(ctx, "Error signing in user")
			ct.JSON(http.StatusInternalServerError, gin.H{"error": "Error signing in user", "internal_error": err.Error()})
			return
		}

		ct.JSON(http.StatusOK, models.NewUserLoginResponse(signIn.Spec.Username, signIn.Spec.Role, until))
		return
	}
}
