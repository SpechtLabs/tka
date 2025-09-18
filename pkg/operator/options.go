// Package operator provides Kubernetes operator functionality for the TKA service.
// This package implements the Kubernetes controller that manages TKASignIn custom
// resources and provisions user credentials within the cluster. It handles the
// lifecycle of authentication credentials and integrates with the Kubernetes API.
package operator

// OperatorOptions holds configuration for the Kubernetes operator behavior and naming.
type OperatorOptions struct {
	Namespace     string
	ClusterName   string
	ContextPrefix string
	UserPrefix    string
}

func defaultOperatorOptions() OperatorOptions {
	return OperatorOptions{
		Namespace:     DefaultNamespace,
		ClusterName:   DefaultClusterName,
		ContextPrefix: DefaultContextPrefix,
		UserPrefix:    DefaultUserEntryPrefix,
	}
}
