package api_test

import (
	"net/http"
	"testing"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/api"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/auth"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/auth/capability"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/auth/mock"
	"github.com/stretchr/testify/require"
)

func TestLogoutHandler(t *testing.T) {
	m := mock.NewMockAuthService()
	_, ts := newTestServer(t, m, capability.Rule{Role: "dev", Period: "10m"})

	tests := []struct {
		name            string
		setup           func()
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "provisioned true -> 200",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidUntil: time.Now().Add(30 * time.Minute).Format(time.RFC3339), Provisioned: true}, nil
				}
				m.LogoutFn = func(string) humane.Error { return nil }
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "not provisioned -> 200 with computed until",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10m", Provisioned: false}, nil
				}
				m.LogoutFn = func(string) humane.Error { return nil }
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "status not found -> 404",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) { return nil, noSigninError }
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "no signin",
		},
		{
			name: "logout error -> 500",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10m", Provisioned: false}, nil
				}
				m.LogoutFn = func(string) humane.Error { return humane.New("fail") }
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "fail",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, body := doReq(t, ts, http.MethodPost, api.ApiRouteV1Alpha1+api.LogoutApiRoute, nil, nil)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			if tc.expectedMessage != "" {
				requireErrorMessage(t, body, tc.expectedMessage)
			}
		})
	}
}
