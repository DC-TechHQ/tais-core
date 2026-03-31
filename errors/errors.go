package errors

import (
	"net/http"
	"sync"

	"github.com/DC-TechHQ/tais-core/i18n"
)

// ── status registry ───────────────────────────────────────────────────────────

var (
	statusMu       sync.RWMutex
	statusRegistry = make(map[string]int)
)

// RegisterStatus maps a domain error code to an HTTP status code.
// Must be called from internal/i18n/ init() functions — never from domain layer.
//
//	pkgerr.RegisterStatus(auth.ErrTokenExpired, http.StatusUnauthorized)
func RegisterStatus(code string, status int) {
	statusMu.Lock()
	defer statusMu.Unlock()
	statusRegistry[code] = status
}

// HTTPStatus returns the registered HTTP status for a code.
// Falls back to 500 if the code has not been registered.
func HTTPStatus(code string) int {
	statusMu.RLock()
	defer statusMu.RUnlock()
	if s, ok := statusRegistry[code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// AppError is the standard error type for all TAIS services.
// Code is an i18n key — the response package translates it to TJ+RU+EN.
// Status is the HTTP status code to return (0 = resolved via RegisterStatus registry).
//
// Never wrap *AppError across layers — it breaks errors.As matching.
type AppError struct {
	Code   string
	Status int
}

func (e *AppError) Error() string {
	return e.Code
}

// New creates a new *AppError with an explicit HTTP status.
// Use for tais-core common errors defined in this package.
func New(code string, status int) *AppError {
	return &AppError{Code: code, Status: status}
}

// NewDomain creates a domain-layer *AppError with no HTTP status embedded.
// The HTTP status is resolved at delivery layer via RegisterStatus.
// Use this in internal/domain/errors.go of each service — keeps domain pure.
//
//	// domain/errors.go
//	const ErrVehicleNotFound = "ErrVehicleNotFound"
//	var ErrVehicleNotFoundErr = pkgerr.NewDomain(ErrVehicleNotFound)
//
//	// internal/i18n/vehicle.go init()
//	pkgerr.RegisterStatus(domain.ErrVehicleNotFound, http.StatusNotFound)
func NewDomain(code string) *AppError {
	return &AppError{Code: code, Status: 0}
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
