package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerr "github.com/DC-TechHQ/tais-core/errors"
	"github.com/DC-TechHQ/tais-core/pagination"
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
	response.OK(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	if body["data"] == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestCreated(t *testing.T) {
	c, w := newContext()
	response.Created(c, map[string]int{"id": 1})

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusCreated)
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
	params := pagination.Params{Page: 2, Limit: 10, Offset: 10}
	response.Paginated(c, []string{"a", "b"}, 25, params)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)

	pg, ok := body["pagination"].(map[string]any)
	if !ok {
		t.Fatal("expected pagination object in response")
	}
	if pg["page"] != float64(2) {
		t.Errorf("pagination.page: got %v, want 2", pg["page"])
	}
	if pg["total"] != float64(25) {
		t.Errorf("pagination.total: got %v, want 25", pg["total"])
	}
	if pg["total_pages"] != float64(3) {
		t.Errorf("pagination.total_pages: got %v, want 3", pg["total_pages"])
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
