package response

import (
	stderrors "errors"
	"math"
	"net/http"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/i18n"
	"github.com/DC-TechHQ/tais-core/pagination"
	"github.com/gin-gonic/gin"
)

// ── envelope types ────────────────────────────────────────────────────────────

// successBody is the standard success envelope.
//
//	{ "success": true, "data": {...} }
type successBody struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// paginatedBody is the standard list + pagination envelope.
//
//	{
//	  "success": true,
//	  "data": [...],
//	  "pagination": { "page": 1, "limit": 20, "total": 500, "total_pages": 25 }
//	}
type paginatedBody struct {
	Success    bool           `json:"success"`
	Data       any            `json:"data"`
	Pagination paginationMeta `json:"pagination"`
}

type paginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// errorBody is the standard error envelope.
//
//	{
//	  "success": false,
//	  "error": {
//	    "code":    "ErrNotFound",
//	    "message": { "tj": "...", "ru": "...", "en": "..." },
//	    "data":    null | { ... }   ← present only when non-nil (e.g. validation errors)
//	  }
//	}
type errorBody struct {
	Success bool        `json:"success"`
	Error   errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string      `json:"code"`
	Message i18nMessage `json:"message"`
	// Data carries additional context for specific error types:
	//   - ErrValidation: []ValidationError
	//   - Business errors: arbitrary struct from the service
	// Omitted from JSON when nil.
	Data any `json:"data,omitempty"`
}

type i18nMessage struct {
	TJ string `json:"tj"`
	RU string `json:"ru"`
	EN string `json:"en"`
}

// ValidationError represents a single field-level validation failure.
// Used together with ErrValidation when returning 400 responses.
//
//	response.ErrorWithData(c, pkgerr.ErrInvalidData, []response.ValidationError{
//	    {Field: "vin", Message: "already exists"},
//	    {Field: "plate_number", Message: "invalid format"},
//	})
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ── public helpers ────────────────────────────────────────────────────────────

// OK responds with HTTP 200 and data wrapped in the success envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, successBody{Success: true, Data: data})
}

// Created responds with HTTP 201 and data wrapped in the success envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, successBody{Success: true, Data: data})
}

// NoContent responds with HTTP 204 (no body).
func NoContent(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}

// Paginated responds with HTTP 200, the data slice, and pagination metadata.
//
//	response.Paginated(c, vehicles, total, params)
func Paginated(c *gin.Context, data any, total int64, params pagination.Params) {
	totalPages := 0
	if params.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}
	c.JSON(http.StatusOK, paginatedBody{
		Success: true,
		Data:    data,
		Pagination: paginationMeta{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Error maps an error to an HTTP status and responds with the trilingual error envelope.
// Handles *pkgerr.AppError natively; falls back to 500 Internal Server Error.
//
//	response.Error(c, err)
func Error(c *gin.Context, err error) {
	ErrorWithData(c, err, nil)
}

// ErrorWithData is the same as Error but attaches extra data to the error envelope.
// Use this for validation errors, business error details, etc.
//
//	response.ErrorWithData(c, pkgerr.ErrInvalidData, []response.ValidationError{
//	    {Field: "vin", Message: "invalid format"},
//	})
func ErrorWithData(c *gin.Context, err error, data any) {
	code := i18n.ErrInternal
	status := http.StatusInternalServerError

	var appErr *pkgerr.AppError
	if stderrors.As(err, &appErr) {
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
