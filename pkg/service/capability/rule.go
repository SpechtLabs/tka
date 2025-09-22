// Package capability provides authentication capability and rule management.
// This package defines the data structures and logic for handling user
// capabilities and access rules that determine what actions users can perform.
package capability

// Rule describes the capability extracted from identity middleware.
type Rule struct {
	// Role is the Kubernetes ClusterRole name to be granted to the user.
	Role string `json:"role"`
	// Period is the duration for which the role is granted.
	Period string `json:"period"`
	// RulePriority is the priority of the rule. Higher priority rules override lower priority rules.
	RulePriority int `json:"priority"`
}

func (r Rule) Priority() int {
	return r.RulePriority
}
