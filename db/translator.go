package db

import (
	"context"
	stderrors "errors"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/logger"
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
//
// Responsibilities:
//   - Logs all non-trivial database errors via log.Error before returning.
//   - Returns nil for nil error and for context.Canceled (graceful cancellation).
//   - Returns pkgerr.ErrNotFound for gorm.ErrRecordNotFound (no logging needed).
//   - Returns typed AppError for known PostgreSQL SQLSTATE codes.
//   - Returns pkgerr.ErrInternal for unrecognised errors.
//
// Usage (from a repository method):
//
//	return database.TranslateError(err, r.log)
func TranslateError(err error, log *logger.Logger) error {
	if err == nil {
		return nil
	}

	// Graceful cancellation — treat as no error; caller decides what to do.
	if stderrors.Is(err, context.Canceled) {
		return nil
	}

	// GORM not-found sentinel — no DB-level error, no logging needed.
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return pkgerr.ErrNotFound
	}

	// PostgreSQL SQLSTATE codes.
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		log.Error("db: sql error",
			"pg_code", pgErr.Code,
			"pg_message", pgErr.Message,
			"pg_detail", pgErr.Detail,
		)
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

	log.Error("db: unhandled error", "error", err)
	return pkgerr.ErrInternal
}
