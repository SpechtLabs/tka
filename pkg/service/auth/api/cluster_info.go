package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getClusterInfo returns the cluster connection information configured for this TKA instance.
// This endpoint exposes the cluster details that authenticated users need to configure their kubeconfig files,
// including the API server endpoint, CA certificate data, TLS settings, and identifying labels.
//
// The returned information is configured via the clusterInfo section in the TKA server configuration
// and represents the externally accessible details of the Kubernetes cluster that users should connect to.
//
// @Summary       Get cluster connection information
// @Description   Returns cluster connection details including API endpoint, CA data, TLS settings, and labels for authenticated users to configure their kubeconfig
// @Tags          authentication
// @Produce       application/json
// @Success       200         {object}  models.TkaClusterInfo     "Successfully returned cluster information"
// @Failure       500         {object}  models.ErrorResponse      "Internal Server Error - Error processing the request"
// @Router        /api/v1alpha1/cluster-info [get]
// @Security      TailscaleAuth
func (t *TKAServer) getClusterInfo(c *gin.Context) {
	req := c.Request
	_, span := t.tracer.Start(req.Context(), "TKAServer.getClusterInfo")
	defer span.End()

	c.JSON(http.StatusOK, t.clusterInfo)
}
