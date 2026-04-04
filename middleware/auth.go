package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pkgctx "github.com/DC-TechHQ/tais-core/context"
	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	pkgjwt "github.com/DC-TechHQ/tais-core/jwt"
	pkgresp "github.com/DC-TechHQ/tais-core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const userCtxTTL = 5 * time.Minute

// UserContextResolver fetches the full user context for an authenticated user.
// Each service implements this interface in infra/resolver/identity.go by calling
// the tais-identity internal endpoint:
//
//	GET {identityURL}/internal/users/{id}/context  (X-Internal-Token header)
//
// The result is cached in Redis under "user_ctx:{user_id}" (TTL 5 min) by the
// Required middleware — implementations should not cache themselves.
type UserContextResolver interface {
	Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error)
}

// Required is the standard JWT authentication middleware for all protected routes.
//
// Flow:
//  1. Extract "Authorization: Bearer {token}"
//  2. Parse and validate JWT (HS256 signature + expiry)
//  3. Check ip_net claim matches client /24 subnet (staff tokens only)
//  4. Check Redis blacklist: tais:blacklist:{jti}  →  401 if found
//  5. Load user context from Redis: user_ctx:{sub}
//     cache miss → resolver.Resolve(ctx, sub) → cache SET user_ctx:{sub} EX 300
//  6. Check is_active = true  →  403 ErrUserBlocked if false
//  7. c.Set(pkgctx.KeyUser, userCtx)
func Required(rdb *redis.Client, jwtCfg pkgjwt.Config, resolver UserContextResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			pkgresp.Error(c, pkgerr.ErrUnauthorized)
			return
		}

		claims, err := pkgjwt.Parse(token, jwtCfg)
		if err != nil {
			pkgresp.Error(c, pkgerr.ErrInvalidToken)
			return
		}

		ctx := c.Request.Context()

		// Blacklist check.
		blacklistKey := fmt.Sprintf("tais:blacklist:%s", claims.JTI)
		exists, err := rdb.Exists(ctx, blacklistKey).Result()
		if err != nil {
			pkgresp.Error(c, pkgerr.ErrInternal)
			return
		}
		if exists > 0 {
			pkgresp.Error(c, pkgerr.ErrInvalidToken)
			return
		}

		// Load user context — Redis first, then resolver on cache miss.
		userCtx, err := loadUserCtx(ctx, rdb, claims.Sub, resolver)
		if err != nil {
			pkgresp.Error(c, pkgerr.ErrUnauthorized)
			return
		}

		// Block check.
		if !userCtx.IsActive {
			pkgresp.Error(c, pkgerr.ErrUserBlocked)
			return
		}

		// IP subnet binding (staff tokens only).
		// super_admin and sysadmin role bypass IP restriction — they may connect from any location.
		if !isBypassIPCheck(userCtx) && !pkgjwt.CheckIPNet(claims, c.ClientIP()) {
			pkgresp.Error(c, pkgerr.ErrUnauthorized)
			return
		}

		pkgctx.SetUser(c, userCtx)
		c.Next()
	}
}

// InternalOnly restricts a route to internal service-to-service calls.
// Compares the X-Internal-Token header against the configured token.
// Mount on the /internal/* route group in router.go.
func InternalOnly(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Internal-Token") != token {
			pkgresp.Error(c, pkgerr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// CitizenOnly aborts with 403 if the authenticated user is not a citizen.
// Must be used after Required.
func CitizenOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := pkgctx.MustGetUser(c)
		if u.Type != string(pkgjwt.TypeCitizen) {
			pkgresp.Error(c, pkgerr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// StaffOnly aborts with 403 if the authenticated user is not a staff member.
// Must be used after Required.
func StaffOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := pkgctx.MustGetUser(c)
		if u.Type != string(pkgjwt.TypeStaff) {
			pkgresp.Error(c, pkgerr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// ── internal helpers ──────────────────────────────────────────────────────────

// loadUserCtx loads user context from Redis cache.
// On cache miss, calls the resolver, then caches the result for 5 minutes.
// Redis errors are treated as cache misses — the resolver is always the source of truth.
func loadUserCtx(
	ctx context.Context,
	rdb *redis.Client,
	userID uint,
	resolver UserContextResolver,
) (*pkgctx.UserCtx, error) {
	cacheKey := fmt.Sprintf("user_ctx:%d", userID)

	val, err := rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var u pkgctx.UserCtx
		if jsonErr := json.Unmarshal(val, &u); jsonErr == nil {
			return &u, nil
		}
	}
	// Redis error or unmarshal failure — fall through to resolver.

	u, err := resolver.Resolve(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Best-effort cache write — never fail the request on cache error.
	if data, jsonErr := json.Marshal(u); jsonErr == nil {
		_ = rdb.Set(ctx, cacheKey, data, userCtxTTL).Err()
	}

	return u, nil
}

func extractBearer(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

// isBypassIPCheck returns true for principals that are allowed to connect from any IP.
// super_admin: system developers — always unrestricted.
// sysadmin: manages VPN infrastructure — must be able to connect remotely to fix issues.
func isBypassIPCheck(u *pkgctx.UserCtx) bool {
	if u.IsSuperAdmin {
		return true
	}
	for _, role := range u.Roles {
		if role == "sysadmin" {
			return true
		}
	}
	return false
}
