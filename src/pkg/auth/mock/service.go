package mock

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/auth"
	"k8s.io/client-go/tools/clientcmd/api"
)

// hand-rolled minimal mock for api.AuthService
type MockAuthService struct {
	SignInFn     func(username, role string, period time.Duration) humane.Error
	StatusFn     func(username string) (*auth.SignInInfo, humane.Error)
	KubeconfigFn func(username string) (*api.Config, humane.Error)
	LogoutFn     func(username string) humane.Error
}

func NewMockAuthService() *MockAuthService {
	return &MockAuthService{}
}

func (m *MockAuthService) SignIn(_ context.Context, username string, role string, period time.Duration) humane.Error {
	if m.SignInFn != nil {
		return m.SignInFn(username, role, period)
	}
	return nil
}

func (m *MockAuthService) Status(_ context.Context, username string) (*auth.SignInInfo, humane.Error) {
	if m.StatusFn != nil {
		return m.StatusFn(username)
	}
	return nil, nil
}

func (m *MockAuthService) Kubeconfig(_ context.Context, username string) (*api.Config, humane.Error) {
	if m.KubeconfigFn != nil {
		return m.KubeconfigFn(username)
	}
	return nil, nil
}

func (m *MockAuthService) Logout(_ context.Context, username string) humane.Error {
	if m.LogoutFn != nil {
		return m.LogoutFn(username)
	}
	return nil
}
