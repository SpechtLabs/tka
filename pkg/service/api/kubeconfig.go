package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	_ "github.com/spechtlabs/tka/pkg/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"
)

// getKubeconfig handles generating and retrieving kubeconfig for authenticated users.
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
//
//nolint:golint-sl // Logs are in mutually exclusive branches (not_ready vs error), only one executes per request
func (t *TKAServer) getKubeconfig(ct *gin.Context) {
	req := ct.Request
	userName := mwauth.GetUsername(ct)

	ctx, span := t.tracer.Start(req.Context(), "TKAServer.getKubeconfig")
	defer span.End()

	// Set initial span attributes
	span.SetAttributes(attribute.String("kubeconfig.username", userName))

	if kubecfg, err := t.client.GetKubeconfig(ctx, userName); err != nil || kubecfg == nil { //nolint:golint-sl // kubecfg used in else branch below
		// Include Retry-After for other async/provisioning flows as a hint
		ct.Header("Retry-After", strconv.Itoa(t.retryAfterSeconds))

		// If the operator indicates credentials are not ready yet, return 202
		if err == k8s.NotReadyYetError {
			span.SetAttributes(
				attribute.String("kubeconfig.status", "not_ready"),
				attribute.Int("kubeconfig.http_status", http.StatusAccepted),
			)
			ct.Status(http.StatusAccepted)
			otelzap.L().InfoContext(ctx, "Kubeconfig not ready yet",
				zap.String("username", userName),
				zap.Int("http_status", http.StatusAccepted),
			)
			return
		}

		// Map NotFound to 401 for this endpoint, otherwise use default mapping
		span.SetAttributes(attribute.String("kubeconfig.status", "error"))
		span.SetStatus(codes.Error, "error getting kubeconfig")
		span.RecordError(err)
		writeHumaneError(ct, err, http.StatusUnauthorized)
		otelzap.L().WithError(err).ErrorContext(ctx, "Error getting kubeconfig")
		return
	} else {
		// Set success attributes
		span.SetAttributes(
			attribute.String("kubeconfig.status", "success"),
			attribute.Int("kubeconfig.http_status", http.StatusOK),
		)

		// Content negotiation: YAML if explicitly requested, otherwise JSON
		if acceptsYAML(ct) {
			span.SetAttributes(attribute.String("kubeconfig.format", "yaml"))
			if data, yerr := yaml.Marshal(kubecfg); yerr == nil {
				ct.Data(http.StatusOK, "application/yaml", data)
				return
			}
			// Fall through to JSON on marshal error
		}
		span.SetAttributes(attribute.String("kubeconfig.format", "json"))
		ct.JSON(http.StatusOK, *kubecfg)
		return
	}
}

// acceptsYAML determines if the client accepts a YAML response based on the Accept header.
// It checks for "application/yaml", "text/yaml", and "application/x-yaml".
func acceptsYAML(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return accept == "application/yaml" || accept == "text/yaml" || accept == "application/x-yaml"
}
