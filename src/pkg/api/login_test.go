package api_test

import (
	"encoding/json"
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

func TestLoginHandler(t *testing.T) {
	m := mock.NewMockAuthService()
	period := "15m"
	rule := capability.Rule{Role: "cluster-admin", Period: period}

	tests := []struct {
		name            string
		rule            capability.Rule
		setup           func()
		expectedStatus  int
		expectedMessage string
	}{
		{
			name: "happy path",
			rule: rule,
			setup: func() {
				m.SignInFn = func(u, r string, d time.Duration) humane.Error {
					require.Equal(t, "alice", u)
					require.Equal(t, "cluster-admin", r)
					require.Equal(t, 15*time.Minute, d)
					return nil
				}
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "no cap rule",
			rule:           capability.Rule{},
			setup:          func() {},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:            "invalid period",
			rule:            capability.Rule{Role: "dev", Period: "garbage"},
			setup:           func() {},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "Error parsing duration",
		},
		{
			name: "signin not found maps to 404",
			rule: rule,
			setup: func() {
				m.SignInFn = func(string, string, time.Duration) humane.Error { return missingError }
			},
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "missing",
		},
		{
			name: "signin generic error maps to 500",
			rule: rule,
			setup: func() {
				m.SignInFn = func(string, string, time.Duration) humane.Error { return humane.New("boom") }
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			_, ts := newTestServer(t, m, tc.rule)
			resp, body := doReq(t, ts, http.MethodPost, api.ApiRouteV1Alpha1+api.LoginApiRoute, nil, map[string]string{})
			require.Equal(t, tc.expectedStatus, resp.StatusCode, string(body))
			if resp.StatusCode == http.StatusAccepted {
				var got struct {
					Until string `json:"until"`
				}
				require.NoError(t, json.Unmarshal(body, &got))
				u, err := time.Parse(time.RFC3339, got.Until)
				require.NoError(t, err)
				require.WithinDuration(t, time.Now().Add(15*time.Minute), u, 2*time.Second)
			} else if tc.expectedMessage != "" {
				requireErrorMessage(t, body, tc.expectedMessage)
			}
		})
	}
}

func TestGetLoginHandler(t *testing.T) {
	m := mock.NewMockAuthService()
	_, ts := newTestServer(t, m, capability.Rule{Role: "dev", Period: "10m"})

	tests := []struct {
		name            string
		setup           func()
		expectedStatus  int
		expectRetry     bool
		expectedMessage string
	}{
		{
			name: "provisioned true -> 200",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidUntil: time.Now().Add(10 * time.Minute).Format(time.RFC3339), Provisioned: true}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "not provisioned -> 202 with Retry-After",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) {
					return &auth.SignInInfo{Username: "alice", Role: "dev", ValidityPeriod: "10m", Provisioned: false}, nil
				}
			},
			expectedStatus: http.StatusAccepted,
			expectRetry:    true,
		},
		{
			name: "not found -> 401",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) { return nil, noSigninError }
			},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "no signin",
		},
		{
			name: "generic error -> 500",
			setup: func() {
				m.StatusFn = func(string) (*auth.SignInInfo, humane.Error) { return nil, humane.New("kaput") }
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "kaput",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, body := doReq(t, ts, http.MethodGet, api.ApiRouteV1Alpha1+api.LoginApiRoute, nil, nil)
			require.Equal(t, tc.expectedStatus, resp.StatusCode)
			if tc.expectRetry {
				require.NotEmpty(t, resp.Header.Get("Retry-After"))
			}
			if tc.expectedMessage != "" {
				requireErrorMessage(t, body, tc.expectedMessage)
			}
		})
	}
}
