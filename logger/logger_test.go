package logger_test

import (
	"testing"

	"github.com/DC-TechHQ/tais-core/logger"
)

func TestNew_Console(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "console",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	log.Info("test message", "key", "value")
	log.With("component", "test").Warn("with fields")
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := logger.New(logger.Config{Level: "verbose"})
	if err == nil {
		t.Error("expected error for unknown level")
	}
}

func TestNew_DefaultLevel(t *testing.T) {
	_, err := logger.New(logger.Config{Format: "console"})
	if err != nil {
		t.Errorf("empty level should default to info, got error: %v", err)
	}
}
