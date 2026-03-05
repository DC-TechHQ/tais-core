package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/DC-TechHQ/tais-core/logger"
	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection parameters.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// New creates and validates a Redis client.
// Returns an error if the connection or PING fails.
func New(cfg Config, log *logger.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	log.Info("redis: connected", "addr", cfg.Addr, "db", cfg.DB)
	return rdb, nil
}
