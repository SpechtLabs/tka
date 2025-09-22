// Package service defines the business logic layer for the TKA service.
// This package provides service interfaces and implementations that handle
// user authentication, Kubernetes credential management, and operator integration.
// It serves as an abstraction layer between the HTTP API and the underlying
// Kubernetes operations.
package k8s

import (
	"context"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// SignInInfo represents the authentication status for a user in a router-agnostic format.
// This structure provides a unified view of user authentication state that abstracts
// away the underlying implementation details (Kubernetes, database, etc.).
type SignInInfo struct {
	// Username is the authenticated user's identity (without domain suffix)
	Username string
	// Role is the user's assigned role (e.g., "admin", "developer", "readonly")
	Role string
	// ValidityPeriod is the original duration requested for credentials (e.g., "24h")
	ValidityPeriod string
	// ValidUntil is the RFC3339 timestamp when credentials expire
	ValidUntil string
	// Provisioned indicates whether credentials are ready for use
	Provisioned bool
}

// TkaClient defines the core business logic operations for user authentication and credential management.
// This interface abstracts the underlying implementation (Kubernetes operator, database, etc.)
// and provides a stable contract for the HTTP API layer.
//
// Implementations should:
// Handle all business logic validation
// Manage credential lifecycle (creation, retrieval, deletion)
// Return humane.Error for structured error handling
// Be safe for concurrent use
type TkaClient interface {
	// SignIn initiates the credential provisioning process for a user.
	// This is an asynchronous operation that may take time to complete.
	NewSignIn(ctx context.Context, username string, role string, period time.Duration) humane.Error

	// Status retrieves the current authentication status for a user.
	// Use this to check if credentials are ready after calling SignIn.
	GetStatus(ctx context.Context, username string) (*SignInInfo, humane.Error)

	// Kubeconfig retrieves the kubeconfig for an authenticated user.
	// This only succeeds if the user has successfully signed in and credentials are provisioned.
	GetKubeconfig(ctx context.Context, username string) (*clientcmdapi.Config, humane.Error)

	// Logout revokes credentials and removes authentication state for a user.
	// This is typically used when users explicitly log out or when cleaning up expired sessions.
	DeleteSignIn(ctx context.Context, username string) humane.Error
}
