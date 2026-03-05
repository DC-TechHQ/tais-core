package db

import (
	"fmt"
	"strings"

	"github.com/DC-TechHQ/tais-core/pagination"
	"gorm.io/gorm"
)

// Builder is a fluent, nil-safe GORM query builder.
// All condition methods skip their clause when the provided value is the zero
// value for its type (empty string, 0, nil slice, etc.), so callers can
// safely forward optional filter fields without extra nil-checks.
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

// Where appends a raw WHERE clause only when cond is a non-empty string or
// when args are supplied.  Wraps gorm.DB.Where — supports positional
// placeholders: Where("status = ?", status).
func (b *Builder) Where(cond string, args ...any) *Builder {
	if cond == "" {
		return b
	}
	// Skip if the only arg is the zero value for its type.
	if len(args) == 1 {
		switch v := args[0].(type) {
		case string:
			if v == "" {
				return b
			}
		case int:
			if v == 0 {
				return b
			}
		case int64:
			if v == 0 {
				return b
			}
		case uint:
			if v == 0 {
				return b
			}
		case uint64:
			if v == 0 {
				return b
			}
		}
	}
	b.db = b.db.Where(cond, args...)
	return b
}

// Search appends a LIKE clause across the given columns when q is non-empty.
// Uses ILIKE on PostgreSQL for case-insensitive matching.
// Example: Search("john", "first_name", "last_name") →
//
//	WHERE (first_name ILIKE '%john%' OR last_name ILIKE '%john%')
func (b *Builder) Search(q string, columns ...string) *Builder {
	if q == "" || len(columns) == 0 {
		return b
	}
	parts := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	pattern := "%" + q + "%"
	for _, col := range columns {
		parts = append(parts, fmt.Sprintf("%s ILIKE ?", col))
		args = append(args, pattern)
	}
	b.db = b.db.Where("("+strings.Join(parts, " OR ")+")", args...)
	return b
}

// DateRange appends created_at / updated_at range filters when from/to are
// non-empty strings in any format accepted by PostgreSQL (ISO 8601 recommended).
func (b *Builder) DateRange(column, from, to string) *Builder {
	if from != "" {
		b.db = b.db.Where(fmt.Sprintf("%s >= ?", column), from)
	}
	if to != "" {
		b.db = b.db.Where(fmt.Sprintf("%s <= ?", column), to)
	}
	return b
}

// OrderBy appends an ORDER BY clause. direction must be "asc" or "desc"
// (case-insensitive); defaults to "asc" for any other value.
func (b *Builder) OrderBy(column, direction string) *Builder {
	if column == "" {
		return b
	}
	dir := "ASC"
	if strings.EqualFold(direction, "desc") {
		dir = "DESC"
	}
	b.db = b.db.Order(fmt.Sprintf("%s %s", column, dir))
	return b
}

// Pagination applies LIMIT and OFFSET from a pagination.Params.
func (b *Builder) Pagination(p pagination.Params) *Builder {
	b.db = b.db.Limit(p.Limit).Offset(p.Offset)
	return b
}
