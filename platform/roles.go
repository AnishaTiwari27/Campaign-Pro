package platform

import "net/http"

// Role constants — a single source of truth shared by every service that
// issues, checks, or assigns a role, so "editor" vs "Editor" typos can't
// drift between auth-service (assigns roles) and campaigns-service
// (checks them). Role itself stays a plain string on the JWT claims
// (platform.Claims.Role) and the trusted X-User-Role header — these
// constants are just the canonical spellings, not a new wire type.
const (
	RoleViewer   = "viewer"
	RoleEditor   = "editor"
	RoleApprover = "approver"
	RoleAdmin    = "admin"
)

var validRoles = map[string]bool{
	RoleViewer:   true,
	RoleEditor:   true,
	RoleApprover: true,
	RoleAdmin:    true,
}

// RoleValid reports whether role is one of the four roles this system
// knows about — checked wherever a role is *assigned* (auth-service's
// handleUpdateUserRole) so a bad value is a clean 400, not a raw DB
// CHECK-constraint violation surfacing as a 500.
func RoleValid(role string) bool {
	return validRoles[role]
}

// RequireRole writes a 403 and returns false unless the request's trusted
// X-User-Role header (set by the gateway after JWT verification — see
// docs/ARCHITECTURE.md) is one of allowed. An empty/missing header (no
// X-User-Role at all) never matches anything, so this fails closed the
// same way the inline checks it replaces always did.
//
// Centralized here — not left as inline per-handler checks — because more
// than one service now needs more-than-one-role gating with different
// allowed-role sets per route; the codebase's own earlier comments
// explicitly flagged "no shared helper, given there are currently only
// two admin-gated routes" as a deliberate choice *for that scale*, not a
// permanent one.
func RequireRole(w http.ResponseWriter, r *http.Request, message string, allowed ...string) bool {
	role := r.Header.Get("X-User-Role")
	for _, a := range allowed {
		if role == a {
			return true
		}
	}
	WriteError(w, http.StatusForbidden, "forbidden", message)
	return false
}
