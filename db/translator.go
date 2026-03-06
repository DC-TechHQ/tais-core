package db

import (
	"context"
	stderrors "errors"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// PostgreSQL error codes (SQLSTATE).
const (
	pgCodeUniqueViolation     = "23505"
	pgCodeForeignKeyViolation = "23503"
	pgCodeNotNullViolation    = "23502"
	pgCodeCheckViolation      = "23514"
	pgCodeDeadlockDetected    = "40P01"
)

// TranslateError maps raw GORM / pgx errors to a clean *pkgerr.AppError.
// Pure translation — no logging. The repository logs the raw error before
// calling this function, so structured context (operation, entity ID, etc.)
// is captured at the point where it is available.
//
// Mapping:
//
//	gorm.ErrRecordNotFound → ErrNotFound      (404)
//	23505 unique_violation  → ErrAlreadyExists (409)
//	23503 foreign_key       → ErrForeignKey    (400)
//	23502 not_null          → ErrInvalidData   (400)
//	23514 check_violation   → ErrInvalidData   (400)
//	40P01 deadlock          → ErrDeadlock      (409)
//	context.Canceled        → nil  (graceful cancellation)
//	everything else         → ErrInternal      (500)
//
// Usage (from a repository method):
//
//	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
//	    r.log.Error("FindByID", "id", id, "error", err)
//	    return nil, pkgdb.TranslateError(err)
//	}
func TranslateError(err error) error {
	if err == nil {
		return nil
	}

	if stderrors.Is(err, context.Canceled) {
		return nil
	}

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerr.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgCodeUniqueViolation:
			return pkgerr.ErrAlreadyExists
		case pgCodeForeignKeyViolation:
			return pkgerr.ErrForeignKey
		case pgCodeNotNullViolation, pgCodeCheckViolation:
			return pkgerr.ErrInvalidData
		case pgCodeDeadlockDetected:
			return pkgerr.ErrDeadlock
		}
		return pkgerr.ErrInternal
	}

	return pkgerr.ErrInternal
}
