package auth

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Middleware abstracts auth integration for the API, enabling tests to inject a mock.
type Middleware interface {
	Use(e *gin.Engine, tracer trace.Tracer)
}
