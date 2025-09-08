package capability

// Rule describes the capability extracted from identity middleware.
type Rule struct {
	Role   string `json:"role"`
	Period string `json:"period"`
}
