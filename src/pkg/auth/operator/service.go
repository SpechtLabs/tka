package operator

import (
	"context"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/auth"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Service implements auth.Service by delegating to operator.KubeOperator.
type Service struct {
	operator *operator.KubeOperator
}

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

func (s *Service) Status(ctx context.Context, username string) (*auth.SignInInfo, humane.Error) {
	signIn, err := s.operator.GetSignInUser(ctx, username)
	if err != nil {
		return nil, err
	}
	return &auth.SignInInfo{
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
