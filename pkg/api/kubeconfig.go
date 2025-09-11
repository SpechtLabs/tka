package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/operator"
	"sigs.k8s.io/yaml"
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
// @Header        202         {integer} Retry-After               "Seconds until next poll recommended"
// @Router        /api/v1alpha1/kubeconfig [get]
// @Security      TailscaleAuth
func (t *TKAServer) getKubeconfig(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getKubeconfig")
	defer span.End()

	if kubecfg, err := t.auth.Kubeconfig(ctx, userName); err != nil || kubecfg == nil {
		// Include Retry-After for other async/provisioning flows as a hint
		ct.Header("Retry-After", strconv.Itoa(t.retryAfterSeconds))

		// If the operator indicates credentials are not ready yet, return 202
		if err == operator.NotReadyYetError {
			ct.Status(http.StatusAccepted)
			otelzap.L().InfoContext(ctx, "Kubeconfig not ready yet")
			return
		}

		// Map NotFound to 401 for this endpoint, otherwise use default mapping
		writeHumaneError(ct, err, http.StatusUnauthorized)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")
		return
	} else {
		// Content negotiation: YAML if explicitly requested, otherwise JSON
		if acceptsYAML(ct) {
			if data, yerr := yaml.Marshal(kubecfg); yerr == nil {
				ct.Data(http.StatusOK, "application/yaml", data)
				return
			}
			// Fall through to JSON on marshal error
		}
		ct.JSON(http.StatusOK, *kubecfg)
		return
	}
}

func acceptsYAML(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	if accept == "application/yaml" || accept == "text/yaml" || accept == "application/x-yaml" {
		return true
	}
	return false
}
