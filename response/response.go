package response

import (
	stderrors "errors"
	"net/http"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/i18n"
	"github.com/gin-gonic/gin"
)

// ── envelope types ────────────────────────────────────────────────────────────

// paginatedMeta is the standard pagination metadata block.
//
//	{ "total": 500, "page": 2, "limit": 20, "total_pages": 25 }
type paginatedMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// errorBody is the standard error envelope.
//
//	{
//	  "success": false,
//	  "error": {
//	    "code":    "ErrNotFound",
//	    "message": { "tj": "...", "ru": "...", "en": "..." }
//	  }
//	}
type errorBody struct {
	Success bool        `json:"success"`
	Error   errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string      `json:"code"`
	Message i18nMessage `json:"message"`
	// Data carries additional context (e.g. validation errors).
	// Omitted from JSON when nil.
	Data any `json:"data,omitempty"`
}

type i18nMessage struct {
	TJ string `json:"tj"`
	RU string `json:"ru"`
	EN string `json:"en"`
}

// ValidationError represents a single field-level validation failure.
// Use together with ErrorWithData for 400 validation responses.
//
//	response.ErrorWithData(c, pkgerr.ErrInvalidData, []response.ValidationError{
//	    {Field: "vin", Message: "already exists"},
//	})
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ── public helpers ────────────────────────────────────────────────────────────

// OK responds with HTTP 200 using a named key for the payload.
//
//	response.OK(c, "user", dto)
//	→ {"success": true, "user": {...}}
func OK(c *gin.Context, key string, data any) {
	c.JSON(http.StatusOK, namedBody(key, data))
}

// Created responds with HTTP 201 using a named key for the payload.
//
//	response.Created(c, "vehicle", dto)
//	→ {"success": true, "vehicle": {...}}
func Created(c *gin.Context, key string, data any) {
	c.JSON(http.StatusCreated, namedBody(key, data))
}

// NoContent responds with HTTP 204 (no body).
func NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

// Paginated responds with HTTP 200, a named list key, and pagination metadata.
//
//	response.Paginated(c, "users", items, total, page, limit)
//	→ {"success": true, "users": [...], "meta": {"total":500,"page":2,"limit":20,"total_pages":25}}
func Paginated(c *gin.Context, key string, data any, total int64, page, limit int) {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	body := gin.H{
		"success": true,
		key:       data,
		"meta": paginatedMeta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}
	c.JSON(http.StatusOK, body)
}

// Error maps an error to an HTTP status and responds with the trilingual error envelope.
// Handles *pkgerr.AppError natively; falls back to 500 Internal Server Error.
func Error(c *gin.Context, err error) {
	ErrorWithData(c, err, nil)
}

// ErrorWithData is the same as Error but attaches extra data to the error envelope.
// Use for validation errors, business error details, etc.
func ErrorWithData(c *gin.Context, err error, data any) {
	code := i18n.ErrInternal
	status := http.StatusInternalServerError

	if appErr, ok := stderrors.AsType[*pkgerr.AppError](err); ok {
		code = appErr.Code
		status = appErr.Status
	}

	c.AbortWithStatusJSON(status, errorBody{
		Success: false,
		Error: errorDetail{
			Code: code,
			Message: i18nMessage{
				TJ: i18n.Get(code, i18n.LangTJ),
				RU: i18n.Get(code, i18n.LangRU),
				EN: i18n.Get(code, i18n.LangEN),
			},
			Data: data,
		},
	})
}

// ── internal helpers ──────────────────────────────────────────────────────────

// namedBody builds a {"success": true, key: data} map for OK/Created responses.
func namedBody(key string, data any) gin.H {
	return gin.H{
		"success": true,
		key:       data,
	}
}
