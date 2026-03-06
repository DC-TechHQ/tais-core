package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/response"
	"github.com/gin-gonic/gin"
)

func newContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestOK(t *testing.T) {
	c, w := newContext()
	response.OK(c, "user", map[string]string{"name": "John"})

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	if body["user"] == nil {
		t.Error("expected 'user' key to be non-nil")
	}
	if body["data"] != nil {
		t.Error("must not contain generic 'data' key")
	}
}

func TestCreated(t *testing.T) {
	c, w := newContext()
	response.Created(c, "vehicle", map[string]int{"id": 42})

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["vehicle"] == nil {
		t.Error("expected 'vehicle' key to be non-nil")
	}
	if body["data"] != nil {
		t.Error("must not contain generic 'data' key")
	}
}

func TestNoContent(t *testing.T) {
	c, w := newContext()
	response.NoContent(c)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestPaginated(t *testing.T) {
	c, w := newContext()
	response.Paginated(c, "users", []string{"a", "b"}, 25, 2, 10)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)

	if body["success"] != true {
		t.Error("expected success=true")
	}
	if body["users"] == nil {
		t.Error("expected 'users' key to be non-nil")
	}
	if body["data"] != nil {
		t.Error("must not contain generic 'data' key")
	}

	meta, ok := body["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta object in response")
	}
	if meta["page"] != float64(2) {
		t.Errorf("meta.page: got %v, want 2", meta["page"])
	}
	if meta["limit"] != float64(10) {
		t.Errorf("meta.limit: got %v, want 10", meta["limit"])
	}
	if meta["total"] != float64(25) {
		t.Errorf("meta.total: got %v, want 25", meta["total"])
	}
	// 25 items / 10 per page = 3 pages (ceil)
	if meta["total_pages"] != float64(3) {
		t.Errorf("meta.total_pages: got %v, want 3", meta["total_pages"])
	}
}

func TestPaginated_NoPaginationKey(t *testing.T) {
	c, w := newContext()
	response.Paginated(c, "items", []any{}, 0, 1, 20)

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)

	if _, exists := body["pagination"]; exists {
		t.Error("response must use 'meta' key, not 'pagination'")
	}
	if _, exists := body["data"]; exists {
		t.Error("response must not contain generic 'data' key")
	}
}

func TestPaginated_TotalPagesCalculation(t *testing.T) {
	cases := []struct {
		total     int64
		limit     int
		wantPages int
	}{
		{total: 0, limit: 20, wantPages: 0},
		{total: 20, limit: 20, wantPages: 1},
		{total: 21, limit: 20, wantPages: 2},
		{total: 100, limit: 20, wantPages: 5},
		{total: 1, limit: 100, wantPages: 1},
	}

	for _, tc := range cases {
		c, w := newContext()
		response.Paginated(c, "items", []any{}, tc.total, 1, tc.limit)

		var body map[string]any
		json.NewDecoder(w.Body).Decode(&body)
		meta := body["meta"].(map[string]any)

		if meta["total_pages"] != float64(tc.wantPages) {
			t.Errorf("total=%d limit=%d: total_pages got %v, want %d",
				tc.total, tc.limit, meta["total_pages"], tc.wantPages)
		}
	}
}

func TestError_AppError(t *testing.T) {
	c, w := newContext()
	response.Error(c, pkgerr.ErrNotFound)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)

	if body["success"] != false {
		t.Error("expected success=false")
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object")
	}
	if errObj["code"] != "ErrNotFound" {
		t.Errorf("code: got %v, want ErrNotFound", errObj["code"])
	}
	msg, ok := errObj["message"].(map[string]any)
	if !ok {
		t.Fatal("expected message object with tj/ru/en")
	}
	if msg["tj"] == "" || msg["ru"] == "" || msg["en"] == "" {
		t.Error("all three language translations must be non-empty")
	}
}

func TestError_UnknownError_Returns500(t *testing.T) {
	c, w := newContext()
	response.Error(c, pkgerr.ErrInternal)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestErrorWithData_ValidationErrors(t *testing.T) {
	c, w := newContext()
	validationErrs := []response.ValidationError{
		{Field: "vin", Message: "invalid format"},
	}
	response.ErrorWithData(c, pkgerr.ErrInvalidData, validationErrs)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	errObj := body["error"].(map[string]any)
	if errObj["data"] == nil {
		t.Error("expected data field in error response")
	}
}
