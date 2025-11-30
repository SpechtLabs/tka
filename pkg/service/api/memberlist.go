package api

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
// It returns the cluster state in either JSON or HTML format based on the Accept header.
// If gossip is not enabled, it returns a 503 Service Unavailable error.
func (t *TKAServer) getMemberlist(c *gin.Context) {
	if t.gossipStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Gossip not enabled"})
		return
	}

	displayData := t.gossipStore.GetDisplayData()

	if wantsJSON(c) {
		c.JSON(http.StatusOK, displayData)
		return
	}

	c.HTML(http.StatusOK, "memberlist.html", displayData)
}

// wantsJSON determines if the client prefers a JSON response based on the Accept header.
func wantsJSON(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "application/json")
}
