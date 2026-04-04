package context

import (
	"github.com/gin-gonic/gin"
)

// KeyUser is the Gin context key under which UserCtx is stored.
// Exported so services can reference it directly if needed.
const KeyUser = "tais_user"

// UserCtx holds the authenticated user's full identity, roles, and permissions.
// Populated by the Required middleware: loaded from Redis cache, or resolved via
// the UserContextResolver on cache miss, then stored back to Redis (TTL 5 min).
type UserCtx struct {
	ID            uint     `json:"id"`
	Type          string   `json:"type"`             // "staff" | "citizen"
	IsSuperAdmin  bool     `json:"is_super_admin"`
	IsActive      bool     `json:"is_active"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	DeptID        *uint    `json:"dept_id,omitempty"`
	RegionID      *uint    `json:"region_id,omitempty"`
	DLAuthorityID *uint    `json:"dl_authority_id,omitempty"`
	IpNet         string   `json:"ip_net,omitempty"` // staff only: "10.200.1"
	JTI           string   `json:"jti"`              // JWT ID — used for blacklist lookup
}

// GetUser returns the UserCtx stored in the Gin context.
// Returns (nil, false) if not populated (unauthenticated route).
func GetUser(c *gin.Context) (*UserCtx, bool) {
	val, exists := c.Get(KeyUser)
	if !exists {
		return nil, false
	}
	u, ok := val.(*UserCtx)
	return u, ok
}

// MustGetUser returns the UserCtx or panics.
// Use only on routes guaranteed to be behind the Required middleware.
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
	c.Set(KeyUser, u)
}

// HasPermission returns true if the user has the given permission code.
// super_admin always passes. The wildcard "*" grants all permissions (admin role).
func HasPermission(u *UserCtx, code string) bool {
	if u.IsSuperAdmin {
		return true
	}
	for _, p := range u.Permissions {
		if p == "*" || p == code {
			return true
		}
	}
	return false
}
