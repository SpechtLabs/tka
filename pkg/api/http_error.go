package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/models"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// WriteHumaneError writes a humane.Error as a JSON models.ErrorResponse with a mapped status code.
// notFoundStatus allows handlers to override the HTTP status for NotFound conditions (e.g., 401 vs 404).
func WriteHumaneError(c *gin.Context, err humane.Error, notFoundStatus int) {
	if err == nil {
		c.Status(http.StatusNoContent)
		return
	}

	status := http.StatusInternalServerError

	if cause := err.Cause(); cause != nil {
		if k8serrors.IsNotFound(cause) {
			if notFoundStatus > 0 {
				status = notFoundStatus
			} else {
				status = http.StatusNotFound
			}
		}
	}

	c.JSON(status, models.FromHumaneError(err))
}
