package main

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterFromKubeconfig(t *testing.T) {
	ca := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")

	tests := []struct {
		name       string
		kubeconfig string
		wantServer string
		wantCA     string
		wantErr    bool
	}{
		{
			name: "cluster with CA",
			kubeconfig: `apiVersion: v1
kind: Config
clusters:
  - name: kubernetes
    cluster:
      server: https://10.0.0.1:6443
      certificate-authority-data: ` + base64.StdEncoding.EncodeToString(ca) + `
`,
			wantServer: "https://10.0.0.1:6443",
			wantCA:     base64.StdEncoding.EncodeToString(ca),
		},
		{
			name: "cluster without CA",
			kubeconfig: `apiVersion: v1
kind: Config
clusters:
  - name: kubernetes
    cluster:
      server: https://10.0.0.1:6443
`,
			wantServer: "https://10.0.0.1:6443",
		},
		{
			name: "no clusters",
			kubeconfig: `apiVersion: v1
kind: Config
`,
		},
		{
			name:       "not a kubeconfig",
			kubeconfig: "clusters: [",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, caData, err := clusterFromKubeconfig(tt.kubeconfig)
			if tt.wantErr {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, tt.wantServer, server)
			assert.Equal(t, tt.wantCA, caData)
		})
	}
}

func TestLoadClusterInfo(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]any
		wantServer string
		wantErr    string
	}{
		{
			name: "explicit endpoint",
			config: map[string]any{
				"clusterInfo.apiEndpoint":           "https://api.example.com:6443",
				"clusterInfo.caData":                "Y2E=",
				"clusterInfo.insecureSkipTLSVerify": true,
			},
			wantServer: "https://api.example.com:6443",
		},
		{
			name:    "no endpoint",
			config:  map[string]any{},
			wantErr: "clusterInfo.apiEndpoint is required",
		},
		{
			name: "endpoint and configMapRef",
			config: map[string]any{
				"clusterInfo.apiEndpoint":          "https://api.example.com:6443",
				"clusterInfo.configMapRef.enabled": true,
			},
			wantErr: "both clusterInfo and configMapRef provided",
		},
		{
			name: "configMapRef without a name",
			config: map[string]any{
				"clusterInfo.configMapRef.enabled":   true,
				"clusterInfo.configMapRef.namespace": "kube-public",
			},
			wantErr: "configMapRef.name and configMapRef.namespace are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			for k, v := range tt.config {
				viper.Set(k, v)
			}

			info, err := loadClusterInfo(context.Background())
			if tt.wantErr != "" {
				require.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, tt.wantServer, info.ServerURL)
			assert.Equal(t, viper.GetString("clusterInfo.caData"), info.CAData)
			assert.Equal(t, viper.GetBool("clusterInfo.insecureSkipTLSVerify"), info.InsecureSkipTLSVerify)
		})
	}
}
