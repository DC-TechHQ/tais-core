package errors_test

import (
	"errors"
	"testing"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
)

func TestAppError_Error(t *testing.T) {
	err := pkgerr.New("ErrSomething", 400)
	if err.Error() != "ErrSomething" {
		t.Errorf("expected %q, got %q", "ErrSomething", err.Error())
	}
}

func TestAppError_ErrorsAs(t *testing.T) {
	var appErr *pkgerr.AppError
	if !errors.As(pkgerr.ErrNotFound, &appErr) {
		t.Error("errors.As must match *AppError")
	}
	if appErr.Status != 404 {
		t.Errorf("expected status 404, got %d", appErr.Status)
	}
}

func TestCommonErrors_NotNil(t *testing.T) {
	errs := []*pkgerr.AppError{
		pkgerr.ErrInternal,
		pkgerr.ErrInvalidData,
		pkgerr.ErrNotFound,
		pkgerr.ErrAlreadyExists,
		pkgerr.ErrForeignKey,
		pkgerr.ErrUnauthorized,
		pkgerr.ErrForbidden,
		pkgerr.ErrInvalidToken,
		pkgerr.ErrTokenExpired,
		pkgerr.ErrUserBlocked,
		pkgerr.ErrInvalidCredentials,
		pkgerr.ErrDeadlock,
	}
	for _, e := range errs {
		if e == nil {
			t.Error("common error var must not be nil")
		}
		if e.Code == "" {
			t.Error("Code must not be empty")
		}
		if e.Status == 0 {
			t.Error("Status must not be zero")
		}
	}
}
