package db

import (
	"fmt"
	"time"

	"github.com/DC-TechHQ/tais-core/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config holds PostgreSQL connection parameters.
// DSN is built by PostgresConfig.DSN() in each service's config/config.go.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // seconds — rotate long-lived connections (default 1800)
	ConnMaxIdleTime int // seconds — reclaim idle connections in quiet periods (default 600)
}

// New opens a GORM connection, configures the connection pool, and sets a slow
// query logger (queries slower than 200ms are logged as warnings).
// Returns an error if the connection or ping fails.
func New(cfg Config, log *logger.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: newGORMLogger(log),
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("db: open connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db: get sql.DB: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = 1800
	}
	idleTime := cfg.ConnMaxIdleTime
	if idleTime <= 0 {
		idleTime = 600
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(lifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(idleTime) * time.Second)

	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping failed: %w", err)
	}

	return db, nil
}
