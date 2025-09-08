package auth

import "github.com/gin-gonic/gin"

const (
	contextKeyUser    = "auth_username"
	contextKeyCapRule = "auth_cap_rule"
)

// SetUsername stores the authenticated username into the context.
func SetUsername(c *gin.Context, username string) {
	c.Set(contextKeyUser, username)
}

// GetUsername retrieves the authenticated username from the context.
func GetUsername(c *gin.Context) string {
	if username, ok := c.Get(contextKeyUser); ok {
		if s, ok := username.(string); ok {
			return s
		}
	}
	return ""
}

// SetCapability stores the capability rule into the context.
func SetCapability[T any](c *gin.Context, rule T) {
	c.Set(contextKeyCapRule, rule)
}

// GetCapability retrieves the capability rule from the context.
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
