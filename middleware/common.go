package middleware

import (
	"fmt"
	"net/http"
	"time"

	pkgctx "github.com/DC-TechHQ/tais-core/context"
	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/logger"
	pkgresp "github.com/DC-TechHQ/tais-core/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Recovery catches panics in handlers, logs them, and returns HTTP 500
// instead of crashing the server process.
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
				)
				pkgresp.Error(c, pkgerr.ErrInternal)
			}
		}()
		c.Next()
	}
}

// RequestLogger logs every HTTP request after it completes.
//
// Fields logged: request_id, method, path, status, latency_ms, client_ip,
// user_agent, user_id (when authenticated).
//
// Log level by status:
//   - 5xx → Error
//   - 4xx → Warn
//   - 2xx/3xx → Info
//
// The generated request_id is stored in the Gin context under "request_id"
// so downstream handlers can forward it for distributed tracing.
func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		status := c.Writer.Status()
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path += "?" + q
		}

		args := []any{
			"request_id", requestID,
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		// Attach user ID when the auth middleware has run successfully.
		if u, ok := pkgctx.GetUser(c); ok {
			args = append(args, "user_id", u.ID)
		}

		if errs := c.Errors.ByType(gin.ErrorTypePrivate).String(); errs != "" {
			args = append(args, "errors", errs)
		}

		switch {
		case status >= http.StatusInternalServerError:
			log.Error("http request", args...)
		case status >= http.StatusBadRequest:
			log.Warn("http request", args...)
		default:
			log.Info("http request", args...)
		}
	}
}

// CORS sets standard CORS response headers for the TAIS API.
// allowedOrigins comes from HTTPConfig.CORSOrigins in each service's config.
// An empty slice permits all origins (development only).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if _, ok := allowed[origin]; ok || len(allowedOrigins) == 0 {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Internal-Token, X-Service-Name, Accept-Language")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
