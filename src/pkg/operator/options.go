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
