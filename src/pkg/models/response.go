package models

// UserLoginResponse represents the response for a successful user login
// @Description Contains authenticated user information and session details
type UserLoginResponse struct {
	// Username of the authenticated user
	// example: alice@example.com
	Username string `json:"username"`

	// Role assigned to the user in Kubernetes
	// example: cluster-admin
	Role string `json:"role"`

	// Expiration timestamp of the authentication credentials in RFC3339 format
	// example: 2023-12-31T23:59:59Z
	Until string `json:"until"`
}

func NewUserLoginResponse(username, role, until string) UserLoginResponse {
	return UserLoginResponse{
		Username: username,
		Role:     role,
		Until:    until,
	}
}
