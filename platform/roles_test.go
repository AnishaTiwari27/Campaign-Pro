package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoleValid(t *testing.T) {
	cases := []struct {
		role string
		want bool
	}{
		{RoleViewer, true},
		{RoleEditor, true},
		{RoleApprover, true},
		{RoleAdmin, true},
		{"", false},
		{"superadmin", false},
		{"Admin", false}, // case-sensitive — no normalization
	}
	for _, c := range cases {
		if got := RoleValid(c.role); got != c.want {
			t.Errorf("RoleValid(%q) = %v, want %v", c.role, got, c.want)
		}
	}
}

func TestRequireRole_AllowsListedRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-User-Role", RoleEditor)
	rr := httptest.NewRecorder()

	if !RequireRole(rr, req, "nope", RoleEditor, RoleAdmin) {
		t.Fatal("RequireRole returned false for an allowed role")
	}
	if rr.Code != http.StatusOK { // recorder defaults to 200 — nothing was written
		t.Fatalf("status = %d, want no response written (200 default)", rr.Code)
	}
}

func TestRequireRole_RejectsUnlistedRole(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-User-Role", RoleViewer)
	rr := httptest.NewRecorder()

	if RequireRole(rr, req, "only editors and admins", RoleEditor, RoleAdmin) {
		t.Fatal("RequireRole returned true for a disallowed role")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}

func TestRequireRole_MissingHeaderFailsClosed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil) // no X-User-Role at all
	rr := httptest.NewRecorder()

	if RequireRole(rr, req, "nope", RoleEditor, RoleAdmin) {
		t.Fatal("RequireRole returned true with no X-User-Role header — should fail closed")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}
