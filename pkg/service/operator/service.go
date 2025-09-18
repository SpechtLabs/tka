package operator

import (
	"context"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/operator"
	"github.com/spechtlabs/tka/pkg/service"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Service implements service.Service by delegating to a Kubernetes operator.
// This is the production implementation that manages TKASignIn custom resources
// in a Kubernetes cluster to handle user authentication and credential provisioning.
//
// The service:
// Creates TKASignIn CRDs for new user sign-ins
// Monitors CRD status to track provisioning progress
// Generates kubeconfigs from provisioned credentials
// Handles credential cleanup on logout
//
// Thread Safety: This implementation is safe for concurrent use.
type Service struct {
	operator *operator.KubeOperator
}

// New creates a new operator-based service implementation.
// This is the primary constructor for production deployments.
func New(op *operator.KubeOperator) *Service {
	return &Service{operator: op}
}

func (s *Service) SignIn(ctx context.Context, username string, role string, period time.Duration) humane.Error {
	if period < operator.MinSigninValidity {
		return humane.New("`period` may not specify a duration less than 10 minutes",
			"Specify a period greater than 10 minutes in your api ACL for user "+username,
		)
	}
	return s.operator.SignInUser(ctx, username, role, period)
}

func (s *Service) Status(ctx context.Context, username string) (*service.SignInInfo, humane.Error) {
	signIn, err := s.operator.GetSignInUser(ctx, username)
	if err != nil {
		return nil, err
	}
	return &service.SignInInfo{
		Username:       signIn.Spec.Username,
		Role:           signIn.Spec.Role,
		ValidityPeriod: signIn.Spec.ValidityPeriod,
		ValidUntil:     signIn.Status.ValidUntil,
		Provisioned:    signIn.Status.Provisioned,
	}, nil
}

func (s *Service) Kubeconfig(ctx context.Context, username string) (*clientcmdapi.Config, humane.Error) {
	return s.operator.GetKubeconfig(ctx, username)
}

func (s *Service) Logout(ctx context.Context, username string) humane.Error {
	return s.operator.LogOutUser(ctx, username)
}
