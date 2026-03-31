package config

import (
	"fmt"
	"os"
	"strings"
)

const secretsPath = "/run/secrets/"

// ReadSecret reads a Docker secret from /run/secrets/{name}.
// Falls back to the environment variable TAIS_{NAME} (uppercased, dashes → underscores).
// Returns an empty string if neither is found — caller decides if it is required.
func ReadSecret(name string) string {
	data, err := os.ReadFile(secretsPath + name)
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	envKey := "TAIS_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	return os.Getenv(envKey)
}

// MustReadSecret calls ReadSecret and panics if the result is empty.
// Use for secrets that are truly required in production (JWT secret, DB password).
func MustReadSecret(name string) string {
	val := ReadSecret(name)
	if val == "" {
		panic(fmt.Sprintf("config: required secret %q not found in %s or env", name, secretsPath))
	}
	return val
}
