package service

// @title Tailscale Kubernetes Auth API
// @version 1.0
// @description API for authenticating and authorizing Kubernetes access via Tailscale identity.
// @contact.name Specht Labs
// @contact.url specht-labs.de
// @contact.email tka@specht-labs.de
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/v1alpha1
// @securityDefinitions.apikey TailscaleAuth
// @in header
// @name X-Tailscale-User
// @description Authentication happens automatically via the Tailscale network. The server performs a WhoIs lookup on the client's IP address to determine identity. This header is for documentation purposes only and is not actually required to be set.

import (
	globalModels "github.com/spechtlabs/tka/pkg/models"
	authModels "github.com/spechtlabs/tka/pkg/service/auth/models"
	orchestratorModels "github.com/spechtlabs/tka/pkg/service/orchestrator/models"
)

// This file ensures all models are included in Swag documentation
var (
	_ = globalModels.ErrorResponse{}
	_ = authModels.UserLoginResponse{}
	_ = orchestratorModels.ClusterListResponse{}
	_ = orchestratorModels.ClusterListItem{}
)
