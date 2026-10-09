// Package identity is the public contract of the identity service: who the
// caller is and what they are allowed to do. Every other service receives
// a User and asks it questions; none of them know how authentication
// works, and none can reach the sessions table.
package identity

import (
	"context"
	"time"
)

// Role is what a user may do. The four the product actually has, as
// opposed to the admin/viewer pair the stub shipped with.
type Role string

const (
	// RoleAdmin manages users and settings as well as campaigns.
	RoleAdmin Role = "admin"
	// RoleApprover can approve and reject campaigns.
	RoleApprover Role = "approver"
	// RoleAnalyst reads everything in scope but decides nothing.
	RoleAnalyst Role = "analyst"
	// RoleClient is an external user: their own account's data only, and
	// never an approval — sign-off is the agency's control, not theirs.
	RoleClient Role = "client"

	// RoleViewer predates the others and behaves as an analyst. Kept so
	// rows written before this migration stay meaningful.
	RoleViewer Role = "viewer"
)

type User struct {
	ID    string
	Email string
	Name  string
	Role  Role
	// IsAgency users see every account. Client users see only the
	// accounts granted to them.
	IsAgency  bool
	CreatedAt time.Time
}

// CanApprove gates every decision endpoint. A client never approves, no
// matter what else is granted to them: approval is the agency signing off
// on spend, so a client approving their own campaign defeats the control.
func (u User) CanApprove() bool {
	return u.IsAgency && (u.Role == RoleAdmin || u.Role == RoleApprover)
}

// CanManageUsers is admin-only.
func (u User) CanManageUsers() bool { return u.IsAgency && u.Role == RoleAdmin }

// IsClient reports whether the caller is external, which decides whether
// benchmarks are suppressed and competitor names stripped.
func (u User) IsClient() bool { return u.Role == RoleClient || !u.IsAgency }

// SessionTTL is how long a login lasts before it must be renewed.
const SessionTTL = 7 * 24 * time.Hour

// Authenticator is what platform's middleware needs: resolve an opaque
// session token to a user, or fail. Declared here so the HTTP layer
// depends on the capability and not on this service.
type Authenticator interface {
	UserForSession(ctx context.Context, token string) (User, error)
}

// DirectoryUser is one account as the user-management screen needs it.
// It lives in this package rather than identity's internals because the
// service that owns the users table has to speak it too, and a sibling
// may read a service's public contract but never reach inside it.
type DirectoryUser struct {
	ID         string
	Email      string
	Name       string
	Role       string
	IsAgency   bool
	CanApprove bool
	CreatedAt  time.Time
}
