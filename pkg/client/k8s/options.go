package k8s

// ClientOptions holds configuration for the Kubernetes operator behavior and naming.
type ClientOptions struct {
	Namespace     string
	ClusterName   string
	ContextPrefix string
	UserPrefix    string
}

// DefaultClientOptions returns ClientOptions with sensible default values for development.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Namespace:     DefaultNamespace,
		ClusterName:   DefaultClusterName,
		ContextPrefix: DefaultContextPrefix,
		UserPrefix:    DefaultUserEntryPrefix,
	}
}
