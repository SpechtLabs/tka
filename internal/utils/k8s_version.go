package utils

import (
	"strconv"

	"github.com/sierrasoftworks/humane-errors-go"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

// IsK8sVerAtLeast checks if the cluster's Kubernetes version is at least the specified
// major.minor version. It queries the Kubernetes API server to determine the current version.
func IsK8sVerAtLeast(majorVersion, minorVersion int) (bool, humane.Error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return false, humane.Wrap(err, "Failed to get Kubernetes config", "ensure you're running inside a Kubernetes cluster or have valid kubeconfig")
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, humane.Wrap(err, "Failed to create discovery client", "check cluster connectivity and authentication")
	}

	versionInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, humane.Wrap(err, "Failed to get Kubernetes version", "check cluster connectivity and API server availability")
	}

	currentMajor, err := strconv.Atoi(versionInfo.Major)
	if err != nil {
		return false, humane.Wrap(err, "Failed to parse Kubernetes major version", "this indicates an unexpected version format from the API server")
	}

	currentMinor, err := strconv.Atoi(versionInfo.Minor)
	if err != nil {
		return false, humane.Wrap(err, "Failed to parse Kubernetes minor version", "this indicates an unexpected version format from the API server")
	}

	// Check if current version is at least the required version
	return (currentMajor > majorVersion) || (currentMajor == majorVersion && currentMinor >= minorVersion), nil
}
