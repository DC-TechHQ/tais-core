package context

import (
	"github.com/gin-gonic/gin"
)

const userCtxKey = "user_ctx"

// UserCtx holds the authenticated user's identity and permissions.
// Populated by the auth middleware via UserContextResolver and cached in Redis.
type UserCtx struct {
	ID          uint     `json:"id"`
	RoleID      uint     `json:"role_id"`
	RoleName    string   `json:"role_name"`
	RegionID    *uint    `json:"region_id,omitempty"`
	DeptID      *uint    `json:"dept_id,omitempty"`
	SuperAdmin  bool     `json:"super_admin"`
	Admin       bool     `json:"admin"`
	Permissions []string `json:"permissions"`
}

// GetUser returns the UserCtx stored in the Gin context.
// Returns (nil, false) if the context is not populated (i.e. unauthenticated route).
func GetUser(c *gin.Context) (*UserCtx, bool) {
	val, exists := c.Get(userCtxKey)
	if !exists {
		return nil, false
	}
	u, ok := val.(*UserCtx)
	return u, ok
}

// MustGetUser returns the UserCtx or panics.
// Use only on routes protected by the auth middleware.
func MustGetUser(c *gin.Context) *UserCtx {
	u, ok := GetUser(c)
	if !ok {
		panic("context: MustGetUser called on unauthenticated request")
	}
	return u
}

// SetUser stores the UserCtx in the Gin context.
// Called exclusively by the auth middleware.
func SetUser(c *gin.Context, u *UserCtx) {
	c.Set(userCtxKey, u)
}

// HasPermission returns true if the user has the given permission code,
// or if the user is super_admin (bypasses all checks),
// or if the user is admin and has the wildcard permission "*".
func HasPermission(u *UserCtx, code string) bool {
	if u.SuperAdmin {
		return true
	}
	for _, p := range u.Permissions {
		if p == "*" || p == code {
			return true
		}
	}
	return false
}
