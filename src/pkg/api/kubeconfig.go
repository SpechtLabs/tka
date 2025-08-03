package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/models"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// getKubeconfig handles generating and retrieving kubeconfig for authenticated users
// @Summary       Get kubeconfig for authenticated user
// @Description   Generates and returns a kubeconfig file for the authenticated Tailscale user
// @Tags          authentication
// @Produce       application/yaml
// @Produce       application/json
// @Success       200         {file}    string                    "OK - Returns kubeconfig file"
// @Failure       400         {object}  models.ErrorResponse      "Bad Request - Tagged nodes not supported or error unmarshaling capability or multiple capability rules"
// @Failure       403         {object}  models.ErrorResponse      "Forbidden - Request from Funnel or no capability rule found"
// @Failure       404         {object}  models.ErrorResponse      "Not Found - User not authenticated or credentials not ready"
// @Failure       500         {object}  models.ErrorResponse      "Internal Server Error - Error with WhoIs or generating kubeconfig"
// @Router        /api/v1alpha1/kubeconfig [get]
// @Security      TailscaleAuth
func (t *TKAServer) getKubeconfig(ct *gin.Context) {
	req := ct.Request
	userName := tailscale.GetTailscaleUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getKubeconfig")
	defer span.End()

	if kubecfg, err := t.operator.GetKubeconfig(ctx, userName); err != nil || kubecfg == nil {
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")

		if errors.Is(err, operator.NotReadyYetError) {
			ct.JSON(http.StatusProcessing, models.FromHumaneError(err))
			return
		}

		if err.Cause() != nil && k8serrors.IsNotFound(err.Cause()) {
			ct.JSON(http.StatusUnauthorized, models.FromHumaneError(err))
			return
		} else {
			ct.JSON(http.StatusInternalServerError, models.FromHumaneError(err))
			return
		}
	} else {
		ct.JSON(http.StatusOK, *kubecfg)
		return
	}
}
