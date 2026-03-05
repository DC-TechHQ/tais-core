package db

import (
	"context"
	"time"

	"github.com/DC-TechHQ/tais-core/logger"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

// gormLogger bridges GORM's logger.Interface to our structured Logger.
//
// Behaviour:
//   - ALL successful queries → Logger.Gorm (gorm.log) at INFO level.
//   - Slow queries (> slowThreshold) → also logged with a "slow_query: true" marker.
//   - Queries with errors → Logger.Error (error.log) — caller will also call TranslateError.
//   - GORM Info/Warn messages → discarded (too noisy, not actionable).
type gormLogger struct {
	log           *logger.Logger
	slowThreshold time.Duration
}

func newGORMLogger(log *logger.Logger) gormlogger.Interface {
	return &gormLogger{log: log, slowThreshold: 200 * time.Millisecond}
}

func (l *gormLogger) LogMode(_ gormlogger.LogLevel) gormlogger.Interface { return l }

// Info and Warn are intentionally no-ops — GORM uses them for internal lifecycle
// messages (e.g. "creating table") that add noise without operational value.
func (l *gormLogger) Info(_ context.Context, _ string, _ ...any) {}
func (l *gormLogger) Warn(_ context.Context, _ string, _ ...any) {}

// Error is called by GORM for driver-level errors.
func (l *gormLogger) Error(_ context.Context, msg string, args ...any) {
	l.log.Error("gorm: "+msg, args...)
}

// Trace is called for every SQL statement executed by GORM.
// This is the primary hook: all queries go to gorm.log; errors and slow queries
// get additional fields to aid debugging and performance analysis.
func (l *gormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		// Log query errors to the error log — TranslateError will map them to AppError.
		l.log.Gorm.Error("gorm: query error",
			zap.Error(err),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("elapsed_ms", elapsed.Milliseconds()),
		)
		return
	}

	slow := elapsed > l.slowThreshold

	// Log every query to gorm.log for full audit trail and debugging.
	l.log.Gorm.Info("gorm: query",
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Int64("elapsed_ms", elapsed.Milliseconds()),
		zap.Bool("slow_query", slow),
	)
}
