// Package api provides HTTP API handlers and routing for the TKA service.
// This package implements REST endpoints for user authentication, Kubernetes
// credential management, and cluster operations. It handles the HTTP layer
// of the TKA service and delegates business logic to the service layer.
package api

import "github.com/spechtlabs/tka/pkg/models"

// This file ensures all models are included in Swag documentation
var (
	_ = models.ErrorResponse{}
	_ = models.UserLoginResponse{}
)
