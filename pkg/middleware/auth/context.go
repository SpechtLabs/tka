package auth

import "github.com/gin-gonic/gin"

const (
	contextKeyUser    = "auth_username"
	contextKeyCapRule = "auth_cap_rule"
)

// SetUsername stores the authenticated username in the Gin context.
// This function is used by authentication middleware to make the username
// available to downstream HTTP handlers.
func SetUsername(c *gin.Context, username string) {
	c.Set(contextKeyUser, username)
}

// GetUsername retrieves the authenticated username from the Gin context.
// This function is used by HTTP handlers to access the current user's identity.
func GetUsername(c *gin.Context) string {
	if username, ok := c.Get(contextKeyUser); ok {
		if s, ok := username.(string); ok {
			return s
		}
	}
	return ""
}

// SetCapability stores a typed capability rule in the Gin context.
// This function is used by authentication middleware to make capability
// information available to downstream HTTP handlers.
func SetCapability[T any](c *gin.Context, rule T) {
	c.Set(contextKeyCapRule, rule)
}

// GetCapability retrieves a typed capability rule from the Gin context.
// This function is used by HTTP handlers to access the current user's permissions.
func GetCapability[T any](c *gin.Context) *T {
	if v, ok := c.Get(contextKeyCapRule); ok {
		if r, ok := v.(T); ok {
			return &r
		}
		if r, ok := v.(*T); ok {
			return r
		}
	}
	return nil
}
