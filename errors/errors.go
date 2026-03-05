package errors

import "github.com/DC-TechHQ/tais-core/i18n"

// AppError is the standard error type for all TAIS services.
// Code is an i18n key — the response package translates it to TJ+RU+EN.
// Status is the HTTP status code to return.
//
// Never wrap *AppError across layers — it breaks errors.As matching.
type AppError struct {
	Code   string
	Status int
}

func (e *AppError) Error() string {
	return e.Code
}

// New creates a new *AppError. Use this to define service-specific error vars.
func New(code string, status int) *AppError {
	return &AppError{Code: code, Status: status}
}

// Common error vars — used across all 28 services.
// Services add domain-specific errors in internal/errors/errors.go.
var (
	ErrInternal           = New(i18n.ErrInternal, 500)
	ErrInvalidData        = New(i18n.ErrInvalidData, 400)
	ErrNotFound           = New(i18n.ErrNotFound, 404)
	ErrAlreadyExists      = New(i18n.ErrAlreadyExists, 409)
	ErrForeignKey         = New(i18n.ErrForeignKey, 400)
	ErrUnauthorized       = New(i18n.ErrUnauthorized, 401)
	ErrForbidden          = New(i18n.ErrForbidden, 403)
	ErrInvalidToken       = New(i18n.ErrInvalidToken, 401)
	ErrTokenExpired       = New(i18n.ErrTokenExpired, 401)
	ErrUserBlocked        = New(i18n.ErrUserBlocked, 403)
	ErrInvalidCredentials = New(i18n.ErrInvalidCredentials, 401)
	ErrDeadlock           = New(i18n.ErrDeadlock, 409)
)
