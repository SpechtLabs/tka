package api_test

import (
	"net/http"
	"testing"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/auth/capability"
	"github.com/spechtlabs/tka/pkg/auth/mock"
	"github.com/stretchr/testify/require"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetKubeconfigHandler(t *testing.T) {
	m := mock.NewMockAuthService()
	_, ts := newTestServer(t, m, capability.Rule{Role: "dev", Period: "10m"})

	cfg := &clientcmdapi.Config{Kind: "Config", APIVersion: "v1", CurrentContext: "x"}

	tests := []struct {
		name            string
		setup           func()
		headers         map[string]string
		expectedStatus  int
		expectRetry     bool
		expectedCT      string
		contains        string
		expectedMessage string
	}{
		{
			name:           "success JSON",
			setup:          func() { m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return cfg, nil } },
			expectedStatus: http.StatusOK,
			expectedCT:     "application/json",
		},
		{
			name:           "success YAML",
			setup:          func() { m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return cfg, nil } },
			headers:        map[string]string{"Accept": "application/yaml"},
			expectedStatus: http.StatusOK,
			expectedCT:     "application/yaml",
			contains:       "kind:",
		},
		{
			name: "not found -> 401",
			setup: func() {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return nil, noSigninError }
			},
			expectedStatus:  http.StatusUnauthorized,
			expectRetry:     true,
			expectedMessage: "no signin",
		},
		{
			name: "generic error -> 500",
			setup: func() {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return nil, humane.New("boom") }
			},
			expectedStatus:  http.StatusInternalServerError,
			expectRetry:     true,
			expectedMessage: "boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, body := doReq(t, ts, http.MethodGet, api.ApiRouteV1Alpha1+api.KubeconfigApiRoute, tc.headers, nil)
			require.Equal(t, tc.expectedStatus, resp.StatusCode, string(body))
			if tc.expectedCT != "" {
				require.Contains(t, resp.Header.Get("Content-Type"), tc.expectedCT)
			}
			if tc.contains != "" {
				require.Contains(t, string(body), tc.contains)
			}
			if tc.expectRetry {
				require.NotEmpty(t, resp.Header.Get("Retry-After"))
			}
			if tc.expectedMessage != "" {
				requireErrorMessage(t, body, tc.expectedMessage)
			}
		})
	}
}
