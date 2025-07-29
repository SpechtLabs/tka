package operator

import (
	"strconv"

	"github.com/sierrasoftworks/humane-errors-go"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func isInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}

// isK8sVerAtLeast checks if the cluster's Kubernetes version is at least the specified major.minor version
func (t *KubeOperator) isK8sVerAtLeast(majorVersion, minorVersion int) (bool, humane.Error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(t.mgr.GetConfig())
	if err != nil {
		return false, humane.Wrap(err, "Failed to create discovery client")
	}

	versionInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, humane.Wrap(err, "Failed to get server version")
	}

	currentMajor, err := strconv.Atoi(versionInfo.Major)
	if err != nil {
		return false, humane.Wrap(err, "Failed to parse Kubernetes major version")
	}

	currentMinor, err := strconv.Atoi(versionInfo.Minor)
	if err != nil {
		return false, humane.Wrap(err, "Failed to parse Kubernetes minor version")
	}

	// Check if current version is at least the required version
	return (currentMajor > majorVersion) || (currentMajor == majorVersion && currentMinor >= minorVersion), nil
}
