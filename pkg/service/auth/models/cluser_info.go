package models

// TkaClusterInfo represents the cluster information that is exposed to users when they query the cluster-info endpoint.
// This information is used by users to configure their kubeconfig files and to understand the cluster they are connecting to.
// @Description Contains cluster information including API endpoint, CA data, TLS settings, and identifying labels
type TkaClusterInfo struct {
	// ServerURL is the public Kubernetes API server URL or IP address that users should connect to.
	// This should be the externally accessible endpoint of the cluster's API server.
	// Example: "https://api.cluster.example.com:6443" or "https://192.168.1.100:6443"
	ServerURL string `json:"server_url"`

	// InsecureSkipTLSVerify controls whether TLS certificate verification should be skipped when connecting to the cluster.
	// When true, the client will accept any certificate presented by the server and any hostname matching errors.
	// This should only be set to true for development/testing environments with self-signed certificates.
	// Production clusters should use valid certificates and keep this false for security.
	InsecureSkipTLSVerify bool `json:"insecure_skip_tls_verify"`

	// CAData contains the base64-encoded Certificate Authority (CA) data for the Kubernetes cluster.
	// This is used to verify the TLS certificate presented by the API server.
	// The data should be the PEM-encoded CA certificate, encoded as base64.
	// If empty and InsecureSkipTLSVerify is false, the system's root CA bundle will be used.
	CAData string `json:"ca_data"`

	// Labels is a set of key-value pairs that can be used to identify and categorize the cluster.
	// These labels help users distinguish between different clusters and can be used for
	// automation, monitoring, or organizational purposes.
	// Common examples: environment (dev/staging/prod), region, project, team ownership, etc.
	Labels map[string]string `json:"labels"`
}
