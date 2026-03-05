package context_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	pkgctx "github.com/DC-TechHQ/tais-core/context"
	"github.com/gin-gonic/gin"
)

func newContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestGetUser_NotSet(t *testing.T) {
	c := newContext()
	u, ok := pkgctx.GetUser(c)
	if ok || u != nil {
		t.Error("expected GetUser to return nil, false on unauthenticated context")
	}
}

func TestSetUser_And_GetUser(t *testing.T) {
	c := newContext()
	want := &pkgctx.UserCtx{ID: 7, RoleName: "operator"}
	pkgctx.SetUser(c, want)

	got, ok := pkgctx.GetUser(c)
	if !ok {
		t.Fatal("expected GetUser to return true after SetUser")
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %d, want %d", got.ID, want.ID)
	}
}

func TestMustGetUser_Panics_WhenNotSet(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustGetUser to panic on unauthenticated context")
		}
	}()
	pkgctx.MustGetUser(newContext())
}

func TestHasPermission_SuperAdmin(t *testing.T) {
	u := &pkgctx.UserCtx{SuperAdmin: true}
	if !pkgctx.HasPermission(u, "vehicle:read") {
		t.Error("super_admin should pass any permission check")
	}
}

func TestHasPermission_Wildcard(t *testing.T) {
	u := &pkgctx.UserCtx{Permissions: []string{"*"}}
	if !pkgctx.HasPermission(u, "vehicle:read") {
		t.Error("wildcard * should match any permission")
	}
}

func TestHasPermission_Exact(t *testing.T) {
	u := &pkgctx.UserCtx{Permissions: []string{"vehicle:read", "vehicle:register"}}
	if !pkgctx.HasPermission(u, "vehicle:read") {
		t.Error("should have vehicle:read")
	}
	if pkgctx.HasPermission(u, "vehicle:dispose") {
		t.Error("should not have vehicle:dispose")
	}
}

func TestHasPermission_NoneGranted(t *testing.T) {
	u := &pkgctx.UserCtx{Permissions: []string{}}
	if pkgctx.HasPermission(u, "vehicle:read") {
		t.Error("user with no permissions should fail all checks")
	}
}
