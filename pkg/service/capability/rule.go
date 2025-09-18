// Package capability provides authentication capability and rule management.
// This package defines the data structures and logic for handling user
// capabilities and access rules that determine what actions users can perform.
package capability

// Rule describes the capability extracted from identity middleware.
type Rule struct {
	Role   string `json:"role"`
	Period string `json:"period"`
}
