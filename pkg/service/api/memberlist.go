package api

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tka/pkg/service/models"
)

// templatesFS embeds the HTML templates located in the templates directory.
//
//go:embed templates/*
var templatesFS embed.FS

// loadTemplates initializes the HTML templates used by the server.
// It defines custom template functions and parses templates from the embedded filesystem.
func (t *TKAServer) loadTemplates() {
	// Create a template with custom functions
	templ := template.New("").Funcs(template.FuncMap{
		"since": func(t time.Time) string {
			return time.Since(t).Round(time.Second).String()
		},
	})

	// Parse templates from the embedded FS
	templ, err := templ.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		// This panic is acceptable as it happens during initialization
		// and indicates a developer error (missing or invalid templates)
		panic(err)
	}

	t.router.SetHTMLTemplate(templ)
}

// getMemberlist handles requests for the gossip memberlist.
// It returns the cluster metadata for all known TKA servers in the mesh.
// If gossip is not enabled, it returns a 503 Service Unavailable error.
//
// @Summary       Get cluster memberlist
// @Description   Returns the list of all clusters known via the gossip protocol. Each entry contains the cluster's API endpoint, TKA port, and labels. Use this endpoint to discover all available clusters in the mesh.
// @Tags          cluster
// @Produce       application/json
// @Produce       text/html
// @Success       200  {array}   models.NodeMetadata   "List of clusters with their connection details"
// @Failure       503  {object}  models.ErrorResponse  "Gossip not enabled on this server"
// @Router        /api/v1alpha1/memberlist [get]
func (t *TKAServer) getMemberlist(c *gin.Context) {
	if t.gossipStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Gossip not enabled"})
		return
	}

	displayData := t.gossipStore.GetDisplayData()

	if wantsJSON(c) {
		// Extract just the cluster metadata from each node
		response := make([]models.NodeMetadata, 0, len(displayData))
		for _, node := range displayData {
			response = append(response, node.State)
		}
		c.JSON(http.StatusOK, response)
		return
	}

	c.HTML(http.StatusOK, "memberlist.html", displayData)
}

// wantsJSON determines if the client prefers a JSON response based on the Accept header.
func wantsJSON(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "application/json")
}
