package api_test

import (
	"net/http"
	"testing"
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/auth"
	"github.com/spechtlabs/tka/pkg/auth/capability"
	"github.com/spechtlabs/tka/pkg/auth/mock"
	"github.com/stretchr/testify/require"
)

func TestLogoutHandler(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(m *mock.MockAuthService) *mock.MockAuthService
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "provisioned true -> 200",
			setup: func(m *mock.MockAuthService) *mock.MockAuthService {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidUntil: time.Now().Add(30 * time.Minute).Format(time.RFC3339), Provisioned: true}, nil
				}
				m.LogoutFn = func(string) humane.Error { return nil }

				return m
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "not provisioned -> 200 with computed until",
			setup: func(m *mock.MockAuthService) *mock.MockAuthService {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10m", Provisioned: false}, nil
				}
				m.LogoutFn = func(string) humane.Error { return nil }

				return m
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "status not found -> 404",
			setup: func(m *mock.MockAuthService) *mock.MockAuthService {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) { return nil, noSigninError }

				return m
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "no signin",
		},
		{
			name: "logout error -> 500",
			setup: func(m *mock.MockAuthService) *mock.MockAuthService {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10m", Provisioned: false}, nil
				}
				m.LogoutFn = func(string) humane.Error { return humane.New("fail") }

				return m
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "fail",
		},
		{
			name: "invalid duration -> 500",
			setup: func(m *mock.MockAuthService) *mock.MockAuthService {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10t", Provisioned: false}, nil
				}
				m.LogoutFn = func(string) humane.Error { return nil }
				return m
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "Error parsing duration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mock.NewMockAuthService()
			_, ts := newTestServer(t, m, capability.Rule{Role: "dev", Period: "10m"})

			tc.setup(m)
			resp, body := doReq(t, ts, http.MethodPost, api.ApiRouteV1Alpha1+api.LogoutApiRoute, nil, nil)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			if tc.expectedMessage != "" {
				requireErrorMessage(t, body, tc.expectedMessage)
			}
		})
	}
}
