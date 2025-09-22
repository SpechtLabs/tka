package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/service/orchestrator/models"
)

// getClusters returns all clusters that the user has access to
// @Summary       List all clusters that the user has access to
// @Description   Lists all clusters that the user has access to
// @Tags          Orchestrator
// @Accept        application/json
// @Produce       application/json
// @Success       200         {object}  models.ClusterListResponse  "A list of clusters that the user has access to"
// @Failure       400         {object}  models.ErrorResponse        "Bad Request - Tagged nodes not supported or error unmarshaling capability or multiple capability rules"
// @Failure       403         {object}  models.ErrorResponse        "Forbidden - Request from Funnel or no capability rule found"
// @Failure       500         {object}  models.ErrorResponse        "Internal Server Error - Error with WhoIs, parsing duration, or signing in user"
// @Router        /orchestrator/v1alpha1/clusters [get]
// @Security      TailscaleAuth
func (t *TKAServer) getClusters(ct *gin.Context) {
	req := ct.Request
	_ = mwauth.GetUsername(ct)

	_, span := t.tracer.Start(req.Context(), "TKAServer.getClusters")
	defer span.End()

	// TODO: replace with actual clusters
	ct.JSON(http.StatusOK, models.NewClusterListResponse(models.ClusterListItem{
		Name:        "prod-east",
		ApiEndpoint: "https://prod-east.tka.specht-labs.de",
		Description: "Production East Cluster",
		Labels:      map[string]string{"environment": "production"},
	}, models.ClusterListItem{
		Name:        "prod-west",
		ApiEndpoint: "https://prod-west.tka.specht-labs.de",
		Description: "Production West Cluster",
		Labels:      map[string]string{"environment": "production"},
	}))
}
