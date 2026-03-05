package middleware

import (
	"context"
	"fmt"
	"strings"

	pkgctx "github.com/DC-TechHQ/tais-core/context"
	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	pkgjwt "github.com/DC-TechHQ/tais-core/jwt"
	pkgresp "github.com/DC-TechHQ/tais-core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// UserContextResolver fetches the full user context for an authenticated user.
// Each service implements this interface in infra/resolver/identity.go by calling
// the tais-identity internal endpoint:
//
//	GET {identityURL}/internal/users/{id}/context  (X-Internal-Token header)
//
// The result is cached in Redis under "user_ctx:{user_id}" (TTL 5 min).
type UserContextResolver interface {
	Resolve(ctx context.Context, userID uint) (*pkgctx.UserCtx, error)
}

// Required is the standard JWT authentication middleware for staff routes.
//
// It:
//  1. Extracts the Bearer token from the Authorization header.
//  2. Parses and validates the JWT signature + expiry.
//  3. Checks the token JTI against the Redis blacklist (tais:blacklist:{jti}).
//  4. Validates the IP subnet binding (ip_net claim vs actual client IP).
//  5. Resolves the full UserCtx via the resolver (with Redis cache).
//  6. Stores UserCtx in the Gin context (pkgctx.SetUser).
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

		// Blacklist check.
		blacklistKey := fmt.Sprintf("tais:blacklist:%s", claims.JTI)
		exists, err := rdb.Exists(c.Request.Context(), blacklistKey).Result()
		if err != nil {
			pkgresp.Error(c, pkgerr.ErrInternal)
			return
		}
		if exists > 0 {
			pkgresp.Error(c, pkgerr.ErrInvalidToken)
			return
		}

		// IP subnet binding (staff tokens only).
		if !pkgjwt.CheckIPNet(claims, c.ClientIP()) {
			pkgresp.Error(c, pkgerr.ErrUnauthorized)
			return
		}

		// Resolve full user context (Redis cache → identity service).
		userCtx, err := resolver.Resolve(c.Request.Context(), claims.Sub)
		if err != nil {
			pkgresp.Error(c, pkgerr.ErrUnauthorized)
			return
		}

		pkgctx.SetUser(c, userCtx)
		c.Next()
	}
}

// InternalOnly restricts a route to internal service-to-service calls.
// Compares the X-Internal-Token header against the configured token.
func InternalOnly(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Internal-Token") != token {
			pkgresp.Error(c, pkgerr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// Can returns a middleware that asserts the authenticated user has a specific
// permission code.  Must be used after Required.
//
//	vehicles.GET("/:id", auth, pkgmw.Can("vehicle:read"), handler.Get)
func Can(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := pkgctx.MustGetUser(c)
		if !pkgctx.HasPermission(u, code) {
			pkgresp.Error(c, pkgerr.ErrForbidden)
			return
		}
		c.Next()
	}
}

// CanAny returns a middleware that asserts the user has at least one of the
// given permission codes.
func CanAny(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := pkgctx.MustGetUser(c)
		for _, code := range codes {
			if pkgctx.HasPermission(u, code) {
				c.Next()
				return
			}
		}
		pkgresp.Error(c, pkgerr.ErrForbidden)
	}
}

// CanAll returns a middleware that asserts the user has ALL of the given
// permission codes.
func CanAll(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := pkgctx.MustGetUser(c)
		for _, code := range codes {
			if !pkgctx.HasPermission(u, code) {
				pkgresp.Error(c, pkgerr.ErrForbidden)
				return
			}
		}
		c.Next()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func extractBearer(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
