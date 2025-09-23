package k8s_test

import (
	"testing"

	"github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestNewKubeconfigWithExternalCluster(t *testing.T) {
	t.Helper()

	contextName := "test-context"
	token := "test-token"
	clusterName := "test-cluster"
	userEntry := "test-user"
	serverURL := "https://external-api.example.com:6443"
	caData := []byte("test-ca-data")
	insecure := false

	config := k8s.NewKubeconfigWithExternalCluster(
		contextName,
		token,
		clusterName,
		userEntry,
		serverURL,
		caData,
		insecure,
	)

	require.NotNil(t, config)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "v1", config.APIVersion)
	assert.Equal(t, contextName, config.CurrentContext)

	// Check cluster configuration
	require.Contains(t, config.Clusters, clusterName)
	cluster := config.Clusters[clusterName]
	assert.Equal(t, serverURL, cluster.Server)
	assert.Equal(t, caData, cluster.CertificateAuthorityData)
	assert.Equal(t, insecure, cluster.InsecureSkipTLSVerify)

	// Check auth info
	require.Contains(t, config.AuthInfos, userEntry)
	authInfo := config.AuthInfos[userEntry]
	assert.Equal(t, token, authInfo.Token)

	// Check context
	require.Contains(t, config.Contexts, contextName)
	context := config.Contexts[contextName]
	assert.Equal(t, clusterName, context.Cluster)
	assert.Equal(t, userEntry, context.AuthInfo)
}

func TestNewKubeconfig_LegacyFunction(t *testing.T) {
	t.Helper()

	contextName := "test-context"
	token := "test-token"
	clusterName := "test-cluster"
	userEntry := "test-user"

	// Create a test rest.Config
	restCfg := &rest.Config{
		Host: "https://internal-api.cluster.local:6443",
	}
	restCfg.CAData = []byte("internal-ca-data")
	restCfg.Insecure = false

	config := k8s.NewKubeconfig(contextName, restCfg, token, clusterName, userEntry)

	require.NotNil(t, config)
	assert.Equal(t, "Config", config.Kind)
	assert.Equal(t, "v1", config.APIVersion)
	assert.Equal(t, contextName, config.CurrentContext)

	// Check cluster configuration uses internal rest.Config values
	require.Contains(t, config.Clusters, clusterName)
	cluster := config.Clusters[clusterName]
	assert.Equal(t, restCfg.Host, cluster.Server)
	assert.Equal(t, restCfg.CAData, cluster.CertificateAuthorityData)
	assert.Equal(t, restCfg.Insecure, cluster.InsecureSkipTLSVerify)

	// Check auth info
	require.Contains(t, config.AuthInfos, userEntry)
	authInfo := config.AuthInfos[userEntry]
	assert.Equal(t, token, authInfo.Token)

	// Check context
	require.Contains(t, config.Contexts, contextName)
	context := config.Contexts[contextName]
	assert.Equal(t, clusterName, context.Cluster)
	assert.Equal(t, userEntry, context.AuthInfo)
}

func TestKubeconfigGeneration_ExternalVsInternal(t *testing.T) {
	t.Helper()

	contextName := "test-context"
	token := "test-token"
	clusterName := "test-cluster"
	userEntry := "test-user"

	// Internal configuration
	restCfg := &rest.Config{
		Host: "https://kubernetes.default.svc.cluster.local",
	}
	restCfg.CAData = []byte("internal-ca-data")
	restCfg.Insecure = false

	// External configuration
	externalServerURL := "https://my-cluster.example.com:6443"
	externalCAData := []byte("external-ca-data")

	internalConfig := k8s.NewKubeconfig(contextName, restCfg, token, clusterName, userEntry)
	externalConfig := k8s.NewKubeconfigWithExternalCluster(
		contextName,
		token,
		clusterName,
		userEntry,
		externalServerURL,
		externalCAData,
		false,
	)

	// Both should have the same basic structure but different cluster endpoints
	assert.Equal(t, internalConfig.Kind, externalConfig.Kind)
	assert.Equal(t, internalConfig.APIVersion, externalConfig.APIVersion)
	assert.Equal(t, internalConfig.CurrentContext, externalConfig.CurrentContext)

	// But different server URLs and CA data
	internalCluster := internalConfig.Clusters[clusterName]
	externalCluster := externalConfig.Clusters[clusterName]

	assert.Equal(t, restCfg.Host, internalCluster.Server)
	assert.Equal(t, externalServerURL, externalCluster.Server)

	assert.Equal(t, restCfg.CAData, internalCluster.CertificateAuthorityData)
	assert.Equal(t, externalCAData, externalCluster.CertificateAuthorityData)

	// Auth info should be identical
	assert.Equal(t, internalConfig.AuthInfos[userEntry].Token, externalConfig.AuthInfos[userEntry].Token)
}

func TestExternalClusterDiscovery_Integration(t *testing.T) {
	t.Helper()

	serverURL := "https://api.example.com:6443"
	caData := []byte("test-ca-data")

	// Test that the discovery mechanism integrates properly with kubeconfig generation
	config := k8s.NewKubeconfigWithExternalCluster(
		"test-context",
		"test-token",
		"test-cluster",
		"test-user",
		serverURL,
		caData,
		false,
	)

	require.NotNil(t, config)
	require.Contains(t, config.Clusters, "test-cluster")

	cluster := config.Clusters["test-cluster"]
	assert.Equal(t, serverURL, cluster.Server)
	assert.Equal(t, caData, cluster.CertificateAuthorityData)
	assert.False(t, cluster.InsecureSkipTLSVerify)
}
