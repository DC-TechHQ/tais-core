package db

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DC-TechHQ/tais-core/pagination"
	"gorm.io/gorm"
)

// columnPattern allows only safe SQL identifiers: letters, digits, underscores, dots.
// Dots are permitted for table-qualified columns: "vehicles.created_at".
var columnPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// Builder is a fluent, nil-safe GORM query builder.
// All condition methods skip their clause when the provided value is the zero
// value for its type (empty string, 0, nil), so callers can safely forward
// optional filter fields without extra nil-checks.
//
// Usage:
//
//	q := pkgdb.NewBuilder(db.WithContext(ctx).Model(&models.Vehicle{})).
//	    Where("status = ?", filter.Status).
//	    Search(filter.Search, "vin", "plate_number").
//	    DateRange("created_at", filter.From, filter.To)
//
//	q.Build().Count(&total)
//	q.Pagination(filter.Params).OrderBy("created_at", "desc").Build().Find(&ms)
type Builder struct {
	db *gorm.DB
}

// NewBuilder wraps a *gorm.DB (already scoped with Model / WithContext) in a Builder.
func NewBuilder(db *gorm.DB) *Builder {
	return &Builder{db: db}
}

// Build returns the underlying *gorm.DB for chaining standard GORM calls
// (Find, Count, First, etc.).
func (b *Builder) Build() *gorm.DB {
	return b.db
}

// Where appends a raw WHERE clause only when the single arg is a non-zero value.
// Supports positional placeholders: Where("status = ?", status).
// Skips the condition when the arg is "", 0, or nil.
func (b *Builder) Where(cond string, args ...any) *Builder {
	if cond == "" {
		return b
	}
	// Skip if the only arg is the zero value for common filter types.
	if len(args) == 1 && isZero(args[0]) {
		return b
	}
	b.db = b.db.Where(cond, args...)
	return b
}

// Search appends a case-insensitive ILIKE clause across the given columns when
// q is non-empty. Only safe column identifiers are accepted.
//
//	Search("john", "first_name", "last_name") →
//	  WHERE (first_name ILIKE '%john%' OR last_name ILIKE '%john%')
func (b *Builder) Search(q string, columns ...string) *Builder {
	if q == "" || len(columns) == 0 {
		return b
	}
	parts := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	pattern := "%" + q + "%"
	for _, col := range columns {
		if !isSafeColumn(col) {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s ILIKE ?", col))
		args = append(args, pattern)
	}
	if len(parts) == 0 {
		return b
	}
	b.db = b.db.Where("("+strings.Join(parts, " OR ")+")", args...)
	return b
}

// DateRange appends >= / <= filters on the given column when from/to are non-empty.
// Accepts ISO 8601 date strings (e.g. "2024-01-01").
func (b *Builder) DateRange(column, from, to string) *Builder {
	if !isSafeColumn(column) {
		return b
	}
	if from != "" {
		b.db = b.db.Where(column+" >= ?", from)
	}
	if to != "" {
		b.db = b.db.Where(column+" <= ?", to)
	}
	return b
}

// OrderBy appends an ORDER BY clause. direction is "asc" or "desc"
// (case-insensitive); defaults to "asc" for any other value.
func (b *Builder) OrderBy(column, direction string) *Builder {
	if !isSafeColumn(column) {
		return b
	}
	dir := "ASC"
	if strings.EqualFold(direction, "desc") {
		dir = "DESC"
	}
	b.db = b.db.Order(column + " " + dir)
	return b
}

// Pagination applies LIMIT and OFFSET from a pagination.Params.
func (b *Builder) Pagination(p pagination.Params) *Builder {
	b.db = b.db.Limit(p.Limit).Offset(p.Offset)
	return b
}

// ── helpers ───────────────────────────────────────────────────────────────────

// isSafeColumn validates that a column name is a safe SQL identifier.
// Prevents SQL injection via column name interpolation.
func isSafeColumn(col string) bool {
	return col != "" && columnPattern.MatchString(col)
}

// isZero reports whether a filter arg should be treated as "not provided".
func isZero(v any) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case uint:
		return val == 0
	case uint32:
		return val == 0
	case uint64:
		return val == 0
	case nil:
		return true
	}
	return false
}
