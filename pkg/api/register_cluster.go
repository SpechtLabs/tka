package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	mwauth "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/models"
)

// registerCluster registers a new cluster
// @Summary       Register a new cluster
// @Description   Registers a new cluster with the orchestrator
// @Tags          Orchestrator
// @Accept        application/json
// @Produce       application/json
// @Param         cluster body models.ClusterListItem true "Cluster to register"
// @Success       201         {object}  models.ClusterListItem     "Successfully registered cluster"
// @Failure       400         {object}  models.ErrorResponse        "Bad Request - Invalid request body or validation errors"
// @Failure       403         {object}  models.ErrorResponse        "Forbidden - Request from Funnel or no capability rule found"
// @Failure       500         {object}  models.ErrorResponse        "Internal Server Error - Error processing the request"
// @Router        /orchestrator/v1alpha1/clusters [post]
// @Security      TailscaleAuth
func (t *TKAServer) registerCluster(ct *gin.Context) {
	req := ct.Request
	_ = mwauth.GetUsername(ct)

	_, span := t.tracer.Start(req.Context(), "TKAServer.registerCluster")
	defer span.End()

	var cluster models.ClusterListItem
	if err := ct.ShouldBindJSON(&cluster); err != nil {
		ct.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// TODO: implement actual cluster registration logic
	// For now, just return the received cluster data
	ct.JSON(http.StatusCreated, cluster)
}
