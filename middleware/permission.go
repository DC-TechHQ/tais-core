package middleware

import (
	pkgctx "github.com/DC-TechHQ/tais-core/context"
	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	pkgresp "github.com/DC-TechHQ/tais-core/response"
	"github.com/gin-gonic/gin"
)

// Can returns a middleware that asserts the authenticated user has a specific
// permission code. Must be used after Required.
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
