package utils

import (
	"strconv"

	"github.com/sierrasoftworks/humane-errors-go"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

// isK8sVerAtLeast checks if the cluster's Kubernetes version is at least the specified major.minor version
func IsK8sVerAtLeast(majorVersion, minorVersion int) (bool, humane.Error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return false, humane.Wrap(err, "Failed to get Kubernetes config")
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, humane.Wrap(err, "Failed to create discovery client")
	}

	versionInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, humane.Wrap(err, "Failed to get tailscale version")
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
