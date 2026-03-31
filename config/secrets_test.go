package config_test

import (
	"os"
	"testing"

	"github.com/DC-TechHQ/tais-core/config"
)

func TestReadSecret_FromEnv(t *testing.T) {
	os.Setenv("TAIS_TEST_SECRET", "secret-value")
	defer os.Unsetenv("TAIS_TEST_SECRET")

	got := config.ReadSecret("test-secret")
	if got != "secret-value" {
		t.Errorf("expected %q, got %q", "secret-value", got)
	}
}

func TestReadSecret_Missing(t *testing.T) {
	got := config.ReadSecret("nonexistent-secret-xyz")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestMustReadSecret_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing secret")
		}
	}()
	config.MustReadSecret("nonexistent-secret-xyz")
}
