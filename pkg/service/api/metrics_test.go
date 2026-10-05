package api_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spechtlabs/tka/pkg/service/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterMetrics(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, reg *prometheus.Registry)
		wantErr bool
	}{
		{
			name:  "fresh registry",
			setup: func(t *testing.T, _ *prometheus.Registry) { t.Helper() },
		},
		{
			name: "registered twice",
			setup: func(t *testing.T, reg *prometheus.Registry) {
				t.Helper()
				require.Nil(t, api.RegisterMetrics(reg))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			tt.setup(t, reg)

			err := api.RegisterMetrics(reg)
			if tt.wantErr {
				require.NotNil(t, err)
				assert.Contains(t, err.Display(), "failed to register the API metrics")
				return
			}
			require.Nil(t, err)
		})
	}
}
