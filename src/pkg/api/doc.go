package api

import "github.com/spechtlabs/tailscale-k8s-auth/pkg/models"

// This file ensures all models are included in Swag documentation
var (
	_ = models.ErrorResponse{}
	_ = models.UserLoginResponse{}
)
