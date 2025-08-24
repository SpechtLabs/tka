package operator

// TODO(cedi): These should be configurable.

// DefaultNamespace is the namespace used for TKA resources when not overridden by configuration.
const DefaultNamespace = "tka-dev"

// DefaultClusterName is the logical name embedded in generated kubeconfigs.
const DefaultClusterName = "tka-cluster"

// DefaultContextPrefix is the prefix for kubeconfig context names.
const DefaultContextPrefix = "tka-context-"

// DefaultUserEntryPrefix is the prefix for kubeconfig user entries.
const DefaultUserEntryPrefix = "tka-user-"
