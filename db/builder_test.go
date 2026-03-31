package db_test

// Builder unit tests verify two things without any database connection:
//
//  1. isSafeColumn — the SQL injection guard rejects unsafe identifiers.
//  2. isZero — the zero-value guard correctly skips empty filter arguments.
//
// Higher-level Builder behaviour (WHERE clauses, ORDER BY, LIMIT/OFFSET) is
// exercised by each service's own integration test suite against a real
// PostgreSQL instance, keeping tais-core free of CGO and network dependencies.

import (
	"testing"

	pkgdb "github.com/DC-TechHQ/tais-core/db"
)

// ── isSafeColumn ──────────────────────────────────────────────────────────────

func TestIsSafeColumn_Valid(t *testing.T) {
	valid := []string{
		"name",
		"created_at",
		"vehicles.created_at",
		"_internal",
		"col123",
		"a",
	}
	for _, col := range valid {
		if !pkgdb.IsSafeColumn(col) {
			t.Errorf("IsSafeColumn(%q) = false, want true", col)
		}
	}
}

func TestIsSafeColumn_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"name; DROP TABLE users--",
		"1starts_with_digit",
		"col with space",
		"col'quoted",
		`col"doublequoted`,
		"col\x00null",
		"(subquery)",
		"col OR 1=1",
	}
	for _, col := range invalid {
		if pkgdb.IsSafeColumn(col) {
			t.Errorf("IsSafeColumn(%q) = true, want false (should be rejected)", col)
		}
	}
}

// ── isZero ────────────────────────────────────────────────────────────────────

func TestIsZero_TrueForZeroValues(t *testing.T) {
	cases := []struct {
		name string
		val  any
	}{
		{"empty string", ""},
		{"int 0", int(0)},
		{"int32 0", int32(0)},
		{"int64 0", int64(0)},
		{"uint 0", uint(0)},
		{"uint32 0", uint32(0)},
		{"uint64 0", uint64(0)},
		{"nil", nil},
	}
	for _, tc := range cases {
		if !pkgdb.IsZero(tc.val) {
			t.Errorf("IsZero(%s) = false, want true", tc.name)
		}
	}
}

func TestIsZero_FalseForNonZeroValues(t *testing.T) {
	cases := []struct {
		name string
		val  any
	}{
		{"non-empty string", "active"},
		{"int 1", int(1)},
		{"int32 1", int32(1)},
		{"int64 1", int64(1)},
		{"uint 1", uint(1)},
		{"uint32 1", uint32(1)},
		{"uint64 1", uint64(1)},
	}
	for _, tc := range cases {
		if pkgdb.IsZero(tc.val) {
			t.Errorf("IsZero(%s) = true, want false", tc.name)
		}
	}
}
