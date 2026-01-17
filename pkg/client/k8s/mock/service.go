package mock

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Compile-time interface verification
var _ k8s.TkaClient = &MockTkaClient{}

// MockTkaClient provides a configurable mock implementation of k8s.TkaClient for testing.
// This allows tests to simulate different scenarios (success, failure, async operations)
// without requiring a real Kubernetes cluster or operator.
//
// Set the function fields to define custom behavior for each method.
// If a function field is nil, the method returns a default success response.
//
// Example:
//
//	mock := &MockTkaClient{
//	  SignInFn: func(username, role string, period time.Duration) humane.Error {
//	    if username == "blocked" {
//	      return humane.New("user blocked", "Contact administrator")
//	    }
//	    return nil
//	  },
//	}
type MockTkaClient struct {
	// SignInFn defines custom behavior for SignIn method calls
	SignInFn func(username, role string, period time.Duration) humane.Error
	// StatusFn defines custom behavior for Status method calls
	StatusFn func(username string) (*k8s.SignInInfo, humane.Error)
	// KubeconfigFn defines custom behavior for Kubeconfig method calls
	KubeconfigFn func(username string) (*api.Config, humane.Error)
	// LogoutFn defines custom behavior for Logout method calls
	LogoutFn func(username string) humane.Error
}

// NewMockTkaClient creates a new mock client with default (success) behavior.
// All methods will return successful responses unless custom functions are configured.
func NewMockTkaClient() k8s.TkaClient {
	return &MockTkaClient{}
}

func (m *MockTkaClient) NewSignIn(_ context.Context, username string, role string, period time.Duration) humane.Error {
	if m.SignInFn != nil {
		return m.SignInFn(username, role, period)
	}
	return nil
}

func (m *MockTkaClient) GetStatus(_ context.Context, username string) (*k8s.SignInInfo, humane.Error) {
	if m.StatusFn != nil {
		return m.StatusFn(username)
	}
	return nil, nil
}

func (m *MockTkaClient) GetKubeconfig(_ context.Context, username string) (*api.Config, humane.Error) {
	if m.KubeconfigFn != nil {
		return m.KubeconfigFn(username)
	}
	return nil, nil
}

func (m *MockTkaClient) DeleteSignIn(_ context.Context, username string) humane.Error {
	if m.LogoutFn != nil {
		return m.LogoutFn(username)
	}
	return nil
}
