package db_test

import (
	"testing"

	"github.com/DC-TechHQ/tais-core/db"
	"github.com/DC-TechHQ/tais-core/pagination"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openSQLite creates an in-memory SQLite DB for Builder tests.
func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return database
}

type testModel struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:100"`
	Status string `gorm:"size:50"`
}

func TestBuilder_Build(t *testing.T) {
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&testModel{Name: "Alice", Status: "active"})
	gdb.Create(&testModel{Name: "Bob", Status: "inactive"})

	var results []testModel
	db.NewBuilder(gdb.Model(&testModel{})).
		Where("status = ?", "active").
		Build().Find(&results)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "Alice" {
		t.Errorf("expected Alice, got %s", results[0].Name)
	}
}

func TestBuilder_Where_SkipsEmpty(t *testing.T) {
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&testModel{Name: "Alice", Status: "active"})
	gdb.Create(&testModel{Name: "Bob", Status: "inactive"})

	var results []testModel
	// Empty string → condition should be skipped → returns all rows.
	db.NewBuilder(gdb.Model(&testModel{})).
		Where("status = ?", "").
		Build().Find(&results)

	if len(results) != 2 {
		t.Errorf("expected 2 results (empty where skipped), got %d", len(results))
	}
}

func TestBuilder_Search(t *testing.T) {
	// Search uses ILIKE (PostgreSQL-specific). This test verifies the builder
	// constructs the query without panicking — full ILIKE behaviour is tested
	// in integration tests against a real PostgreSQL instance.
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&testModel{Name: "Alice"})
	gdb.Create(&testModel{Name: "Bob"})

	// Just verify Search returns a non-nil builder without panic.
	b := db.NewBuilder(gdb.Model(&testModel{})).Search("ali", "name")
	if b == nil {
		t.Error("Search returned nil builder")
	}
}

func TestBuilder_Search_EmptySkips(t *testing.T) {
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&testModel{Name: "Alice"})
	gdb.Create(&testModel{Name: "Bob"})

	var results []testModel
	db.NewBuilder(gdb.Model(&testModel{})).
		Search("", "name").
		Build().Find(&results)

	if len(results) != 2 {
		t.Errorf("expected 2 results (empty search skipped), got %d", len(results))
	}
}

func TestBuilder_OrderBy(t *testing.T) {
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&testModel{Name: "Charlie"})
	gdb.Create(&testModel{Name: "Alice"})
	gdb.Create(&testModel{Name: "Bob"})

	var results []testModel
	db.NewBuilder(gdb.Model(&testModel{})).
		OrderBy("name", "asc").
		Build().Find(&results)

	if results[0].Name != "Alice" {
		t.Errorf("expected Alice first after asc sort, got %s", results[0].Name)
	}
}

func TestBuilder_Pagination(t *testing.T) {
	gdb := openSQLite(t)
	if err := gdb.AutoMigrate(&testModel{}); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		gdb.Create(&testModel{Name: "item"})
	}

	params := pagination.Params{Page: 2, Limit: 3, Offset: 3}
	var results []testModel
	db.NewBuilder(gdb.Model(&testModel{})).
		Pagination(params).
		Build().Find(&results)

	if len(results) != 3 {
		t.Errorf("expected 3 results (page 2, limit 3), got %d", len(results))
	}
}

func TestBuilder_DateRange(t *testing.T) {
	gdb := openSQLite(t)
	// DateRange applies raw WHERE clauses — verify it doesn't panic.
	b := db.NewBuilder(gdb.Model(&testModel{})).
		DateRange("created_at", "2024-01-01", "2024-12-31")
	if b == nil {
		t.Error("DateRange returned nil builder")
	}
}
