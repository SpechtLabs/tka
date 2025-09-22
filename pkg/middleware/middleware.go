// Package middleware provides HTTP middleware components for the TKA service.
// This package contains reusable middleware functionality that can be applied
// to HTTP handlers, including authentication and other cross-cutting concerns.
package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Middleware abstracts auth integration for the API, enabling tests to inject a mock.
type Middleware interface {
	Use(e *gin.Engine, tracer trace.Tracer)
	UseGroup(rg *gin.RouterGroup, tracer trace.Tracer)
}
