package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DC-TechHQ/tais-core/db"
	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TestTranslateError_Nil(t *testing.T) {
	if err := db.TranslateError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestTranslateError_ContextCanceled(t *testing.T) {
	if err := db.TranslateError(context.Canceled); err != nil {
		t.Errorf("context.Canceled should return nil, got %v", err)
	}
}

func TestTranslateError_NotFound(t *testing.T) {
	err := db.TranslateError(gorm.ErrRecordNotFound)
	assertAppError(t, err, pkgerr.ErrNotFound)
}

func TestTranslateError_UniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	err := db.TranslateError(pgErr)
	assertAppError(t, err, pkgerr.ErrAlreadyExists)
}

func TestTranslateError_ForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503", Message: "foreign key violation"}
	err := db.TranslateError(pgErr)
	assertAppError(t, err, pkgerr.ErrForeignKey)
}

func TestTranslateError_CheckViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23514", Message: "check violation"}
	err := db.TranslateError(pgErr)
	assertAppError(t, err, pkgerr.ErrInvalidData)
}

func TestTranslateError_NotNullViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23502", Message: "not null violation"}
	err := db.TranslateError(pgErr)
	assertAppError(t, err, pkgerr.ErrInvalidData)
}

func TestTranslateError_Deadlock(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}
	err := db.TranslateError(pgErr)
	assertAppError(t, err, pkgerr.ErrDeadlock)
}

func TestTranslateError_UnknownError(t *testing.T) {
	err := db.TranslateError(errors.New("some unknown error"))
	assertAppError(t, err, pkgerr.ErrInternal)
}

func assertAppError(t *testing.T, got error, want *pkgerr.AppError) {
	t.Helper()
	var appErr *pkgerr.AppError
	if !errors.As(got, &appErr) {
		t.Fatalf("expected *AppError, got %T: %v", got, got)
	}
	if appErr.Code != want.Code {
		t.Errorf("Code: got %q, want %q", appErr.Code, want.Code)
	}
	if appErr.Status != want.Status {
		t.Errorf("Status: got %d, want %d", appErr.Status, want.Status)
	}
}
