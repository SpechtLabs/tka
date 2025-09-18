package models

// ClusterListResponse represents the response for a list of clusters
// @Description Contains a list of clusters that the user has access to
type ClusterListResponse struct {
	Items []ClusterListItem `json:"items"`
}

// ClusterListItem represents a single cluster in the list
// @Description Contains the details of a single cluster
type ClusterListItem struct {
	Name        string            `json:"name"`
	ApiEndpoint string            `json:"api_endpoint"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// NewClusterListResponse creates a new ClusterListResponse with the provided cluster items.
// This constructor supports variadic arguments for convenient list creation.
func NewClusterListResponse(items ...ClusterListItem) ClusterListResponse {
	return ClusterListResponse{
		Items: items,
	}
}
