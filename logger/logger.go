package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger configuration.
//
// JSON format (production):
//
//	Writes structured JSON to rotating per-level files inside Directory
//	AND to stdout so Docker log drivers / Grafana Loki pick it up automatically.
//
//	  {Directory}/info.log   — info + debug
//	  {Directory}/warn.log   — warnings
//	  {Directory}/error.log  — errors
//	  {Directory}/gorm.log   — slow DB queries (managed by db/gorm_logger.go)
//
// Console format (development):
//
//	Human-readable coloured output to stdout only. No files.
type Config struct {
	Directory  string // e.g. "/var/log/tais-vehicle" — required in JSON mode
	Level      string // "debug" | "info" | "warn" | "error" — defaults to "info"
	Format     string // "json" | "console" — defaults to "json"
	MaxSizeMB  int    // max size per file before rotation (MB) — defaults to 100
	MaxBackups int    // old rotated files to keep — defaults to 10
	MaxAgeDays int    // max age of old files in days — defaults to 30
	Compress   bool   // gzip-compress rotated files
}

// Logger provides structured, levelled logging via zap.
// Each service creates ONE Logger at startup and passes it to sub-components.
// Use With() to create scoped child loggers (e.g. per handler or repository).
type Logger struct {
	info  *zap.Logger
	warn  *zap.Logger
	err   *zap.Logger
	debug *zap.Logger

	// Gorm is a raw *zap.Logger exposed for the db/gorm_logger adapter.
	// Application code should use Info/Warn/Error/Debug methods instead.
	Gorm *zap.Logger
}

// New builds and returns a Logger.
// Returns an error if the log directory cannot be created or the level is unknown.
func New(cfg Config) (*Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	applyDefaults(&cfg)

	if cfg.Format != "json" {
		return newConsoleLogger(level), nil
	}

	if cfg.Directory == "" {
		return nil, fmt.Errorf("logger: Directory is required for JSON format")
	}
	if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
		return nil, fmt.Errorf("logger: create log directory %q: %w", cfg.Directory, err)
	}
	return newFileLogger(cfg, level), nil
}

// Info logs a message at INFO level with optional key-value pairs.
//
//	log.Info("user created", "id", user.ID, "email", user.Email)
func (l *Logger) Info(msg string, args ...any) {
	l.info.Info(msg, toFields(args)...)
}

// Warn logs a message at WARN level with optional key-value pairs.
func (l *Logger) Warn(msg string, args ...any) {
	l.warn.Warn(msg, toFields(args)...)
}

// Error logs a message at ERROR level with optional key-value pairs.
//
//	log.Error("db query failed", "error", err, "id", id)
func (l *Logger) Error(msg string, args ...any) {
	l.err.Error(msg, toFields(args)...)
}

// Debug logs a message at DEBUG level with optional key-value pairs.
func (l *Logger) Debug(msg string, args ...any) {
	l.debug.Debug(msg, toFields(args)...)
}

// Fatal logs a message at ERROR level then calls os.Exit(1).
func (l *Logger) Fatal(msg string, args ...any) {
	l.err.Fatal(msg, toFields(args)...)
}

// With returns a child Logger with the given key-value pairs permanently
// attached to every subsequent call. Safe to call concurrently.
//
//	repoLog := log.With("component", "vehicle-repo")
//	repoLog.Error("FindByID failed", "error", err, "id", id)
func (l *Logger) With(args ...any) *Logger {
	fields := toFields(args)
	return &Logger{
		info:  l.info.With(fields...),
		warn:  l.warn.With(fields...),
		err:   l.err.With(fields...),
		debug: l.debug.With(fields...),
		Gorm:  l.Gorm.With(fields...),
	}
}

// Sync flushes any buffered log entries. Always call on graceful shutdown.
func (l *Logger) Sync() {
	_ = l.info.Sync()
	_ = l.warn.Sync()
	_ = l.err.Sync()
	_ = l.debug.Sync()
	_ = l.Gorm.Sync()
}

// ── internal constructors ─────────────────────────────────────────────────────

func newConsoleLogger(level zapcore.Level) *Logger {
	enc := zapcore.NewConsoleEncoder(devEncoderConfig())
	out := newStdoutSyncer()

	build := func(lvl zapcore.Level) *zap.Logger {
		return zap.New(
			zapcore.NewCore(enc, out, lvl),
			zap.AddCaller(), zap.AddCallerSkip(1),
		)
	}
	return &Logger{
		info:  build(level),
		warn:  build(zapcore.WarnLevel),
		err:   build(zapcore.ErrorLevel),
		debug: build(zapcore.DebugLevel),
		Gorm:  build(zapcore.InfoLevel),
	}
}

func newFileLogger(cfg Config, level zapcore.Level) *Logger {
	jsonEnc := zapcore.NewJSONEncoder(prodEncoderConfig())
	stdout := newStdoutSyncer()

	// rotatingFile returns a lumberjack writer for the given filename.
	rotatingFile := func(name string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(cfg.Directory, name),
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
			LocalTime:  true,
		})
	}

	// teeLogger writes to both the rotating file and stdout.
	// AddCallerSkip(1) skips the Logger method wrapper so logs show actual caller.
	teeLogger := func(filename string, lvl zapcore.Level) *zap.Logger {
		core := zapcore.NewTee(
			zapcore.NewCore(jsonEnc, rotatingFile(filename), lvl),
			zapcore.NewCore(jsonEnc, stdout, lvl),
		)
		return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	gormLogger := zap.New(
		zapcore.NewCore(jsonEnc, rotatingFile("gorm.log"), zapcore.InfoLevel),
		zap.AddCaller(), zap.AddCallerSkip(1),
	)

	return &Logger{
		info:  teeLogger("info.log", level),
		warn:  teeLogger("warn.log", zapcore.WarnLevel),
		err:   teeLogger("error.log", zapcore.ErrorLevel),
		debug: teeLogger("debug.log", zapcore.DebugLevel),
		Gorm:  gormLogger,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// toFields converts variadic key-value pairs to []zap.Field.
// Keys must be strings. Odd-length args append a "!extra" field for the stray value.
func toFields(args []any) []zap.Field {
	if len(args) == 0 {
		return nil
	}
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = fmt.Sprintf("!key%d", i)
		}
		fields = append(fields, zap.Any(key, args[i+1]))
	}
	if len(args)%2 != 0 {
		fields = append(fields, zap.Any("!extra", args[len(args)-1]))
	}
	return fields
}

func parseLevel(s string) (zapcore.Level, error) {
	switch strings.ToLower(s) {
	case "", "info":
		return zapcore.InfoLevel, nil
	case "debug":
		return zapcore.DebugLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return 0, fmt.Errorf("logger: unknown level %q (valid: debug|info|warn|error)", s)
	}
}

func applyDefaults(cfg *Config) {
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 100
	}
	if cfg.MaxBackups <= 0 {
		cfg.MaxBackups = 10
	}
	if cfg.MaxAgeDays <= 0 {
		cfg.MaxAgeDays = 30
	}
}

func prodEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewProductionEncoderConfig()
	ec.TimeKey = "ts"
	ec.EncodeTime = zapcore.ISO8601TimeEncoder
	ec.EncodeLevel = zapcore.LowercaseLevelEncoder
	return ec
}

func devEncoderConfig() zapcore.EncoderConfig {
	ec := zap.NewDevelopmentEncoderConfig()
	ec.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	return ec
}
