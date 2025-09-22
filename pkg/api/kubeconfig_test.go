package api_test

import (
	"net/http"
	"testing"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/api"
	client "github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/spechtlabs/tka/pkg/client/k8s/mock"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/stretchr/testify/require"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetKubeconfigHandler(t *testing.T) {
	m := mock.NewMockTkaClient()
	_, ts := newTestServer(t, m, capability.Rule{Role: "dev", Period: "10m"})

	cfg := &clientcmdapi.Config{Kind: "Config", APIVersion: "v1", CurrentContext: "x"}

	tests := []struct {
		name            string
		setup           func(m *mock.MockTkaClient) client.TkaClient
		headers         map[string]string
		expectedStatus  int
		expectRetry     bool
		expectedCT      string
		contains        string
		expectedMessage string
	}{
		{
			name: "success JSON",
			setup: func(m *mock.MockTkaClient) client.TkaClient {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return cfg, nil }
				return m
			},
			expectedStatus: http.StatusOK,
			expectedCT:     "application/json",
		},
		{
			name: "success YAML",
			setup: func(m *mock.MockTkaClient) client.TkaClient {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return cfg, nil }
				return m
			},
			headers:        map[string]string{"Accept": "application/yaml"},
			expectedStatus: http.StatusOK,
			expectedCT:     "application/yaml",
			contains:       "kind:",
		},
		{
			name: "not found -> 401",
			setup: func(m *mock.MockTkaClient) client.TkaClient {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return nil, noSigninError }
				return m
			},
			expectedStatus:  http.StatusUnauthorized,
			expectRetry:     true,
			expectedMessage: "no signin",
		},
		{
			name: "generic error -> 500",
			setup: func(m *mock.MockTkaClient) client.TkaClient {
				m.KubeconfigFn = func(string) (*clientcmdapi.Config, humane.Error) { return nil, humane.New("boom") }
				return m
			},
			expectedStatus:  http.StatusInternalServerError,
			expectRetry:     true,
			expectedMessage: "boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(m.(*mock.MockTkaClient))
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
