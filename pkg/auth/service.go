package auth

import (
	"context"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// SignInInfo represents the authentication status for a user in a router-agnostic format.
type SignInInfo struct {
	Username       string
	Role           string
	ValidityPeriod string
	ValidUntil     string
	Provisioned    bool
}

// Service defines the operations required by API handlers. Implementations should
// encapsulate any persistence/Kubernetes-specific details and expose a stable contract
// for the HTTP layer.
type Service interface {
	SignIn(ctx context.Context, username string, role string, period time.Duration) humane.Error
	Status(ctx context.Context, username string) (*SignInInfo, humane.Error)
	Kubeconfig(ctx context.Context, username string) (*clientcmdapi.Config, humane.Error)
	Logout(ctx context.Context, username string) humane.Error
}
