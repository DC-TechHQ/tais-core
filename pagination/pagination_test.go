package pagination_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DC-TechHQ/tais-core/pagination"
	"github.com/gin-gonic/gin"
)

func newContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/?"+query, nil)
	c.Request = req
	return c
}

func TestParse_Defaults(t *testing.T) {
	p := pagination.Parse(newContext(""))
	if p.Page != 1 {
		t.Errorf("Page: got %d, want 1", p.Page)
	}
	if p.Limit != 20 {
		t.Errorf("Limit: got %d, want 20", p.Limit)
	}
	if p.Offset != 0 {
		t.Errorf("Offset: got %d, want 0", p.Offset)
	}
}

func TestParse_Custom(t *testing.T) {
	p := pagination.Parse(newContext("page=3&limit=10"))
	if p.Page != 3 {
		t.Errorf("Page: got %d, want 3", p.Page)
	}
	if p.Limit != 10 {
		t.Errorf("Limit: got %d, want 10", p.Limit)
	}
	if p.Offset != 20 {
		t.Errorf("Offset: got %d, want 20", p.Offset)
	}
}

func TestParse_LimitCappedAt100(t *testing.T) {
	p := pagination.Parse(newContext("limit=500"))
	if p.Limit != 100 {
		t.Errorf("Limit should be capped at 100, got %d", p.Limit)
	}
}

func TestParse_InvalidFallsToDefaults(t *testing.T) {
	p := pagination.Parse(newContext("page=abc&limit=-5"))
	if p.Page != 1 {
		t.Errorf("Invalid page should default to 1, got %d", p.Page)
	}
	if p.Limit != 20 {
		t.Errorf("Invalid limit should default to 20, got %d", p.Limit)
	}
}
