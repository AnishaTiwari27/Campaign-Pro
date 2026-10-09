package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/identity"
)

// ErrNotPermitted is returned when the caller is authenticated but is not
// an admin. Distinct from a failed sign-in: the request is well formed and
// the person is who they say they are, they simply may not do this.
var ErrNotPermitted = httpx.ErrForbidden

// ManageableRoles are the roles an admin may assign. Viewer is absent on
// purpose: it predates the others and is kept only so old rows stay
// readable, so nothing new should be created as one.
var ManageableRoles = []identity.Role{
	identity.RoleAdmin, identity.RoleApprover, identity.RoleAnalyst, identity.RoleClient,
}

func validManagedRole(role string) bool {
	for _, r := range ManageableRoles {
		if role == string(r) {
			return true
		}
	}
	return false
}

// Users is the roster, for an admin. Not filtered: the point of the screen
// is seeing everyone who can sign in, including the accounts nobody
// remembers creating.
func (a *Auth) Users(ctx context.Context, actor identity.User) ([]identity.DirectoryUser, error) {
	if !actor.CanManageUsers() {
		return nil, ErrNotPermitted
	}
	if a.directory == nil {
		return nil, fmt.Errorf("user management is not configured on this server")
	}
	return a.directory.ListUsers(ctx)
}

// NewUserInput is what the create form collects. No password: one is
// generated and shown once, because an admin typing a password for someone
// else means that password exists in two heads and usually in a chat log.
type NewUserInput struct {
	Name     string
	Email    string
	Role     string
	IsAgency bool
}

// CreateUser adds an account and returns it with the generated password,
// which is the only time that password is ever available.
func (a *Auth) CreateUser(ctx context.Context, actor identity.User, in NewUserInput) (identity.DirectoryUser, string, error) {
	if !actor.CanManageUsers() {
		return identity.DirectoryUser{}, "", ErrNotPermitted
	}
	if a.directory == nil {
		return identity.DirectoryUser{}, "", fmt.Errorf("user management is not configured on this server")
	}
	clean, err := validateSignup(SignupInput{Name: in.Name, Email: in.Email, Password: "placeholder-only"})
	if err != nil {
		return identity.DirectoryUser{}, "", err
	}
	if !validManagedRole(in.Role) {
		return identity.DirectoryUser{}, "", validationError{field: "role", msg: "pick one of admin, approver, analyst or client"}
	}

	taken, err := a.directory.EmailTaken(ctx, clean.Email)
	if err != nil {
		return identity.DirectoryUser{}, "", fmt.Errorf("check email: %w", err)
	}
	if taken {
		return identity.DirectoryUser{}, "", ErrEmailTaken
	}

	password, err := generatePassword()
	if err != nil {
		return identity.DirectoryUser{}, "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return identity.DirectoryUser{}, "", err
	}

	id, err := a.directory.CreateUser(ctx, clean.Email, clean.Name, in.Role, in.IsAgency)
	if err != nil {
		return identity.DirectoryUser{}, "", fmt.Errorf("create user: %w", err)
	}
	if err := a.store.SetPassword(ctx, id, hash); err != nil {
		return identity.DirectoryUser{}, "", fmt.Errorf("set password: %w", err)
	}

	_ = a.auditor.Record(ctx, actor.ID, actor.Name, "Created account "+clean.Email, "user", id)
	return identity.DirectoryUser{
		ID: id, Email: clean.Email, Name: clean.Name, Role: in.Role,
		IsAgency: in.IsAgency,
		CanApprove: in.IsAgency &&
			(in.Role == string(identity.RoleAdmin) || in.Role == string(identity.RoleApprover)),
	}, password, nil
}

// SetRole changes what an account may do.
//
// An admin cannot change their own role. Not paternalism: an admin who
// demotes themselves by accident locks the last way back into the system,
// and this product has no password-reset flow or second admin guarantee to
// recover from it.
func (a *Auth) SetRole(ctx context.Context, actor identity.User, userID, role string, isAgency bool) error {
	if !actor.CanManageUsers() {
		return ErrNotPermitted
	}
	if a.directory == nil {
		return fmt.Errorf("user management is not configured on this server")
	}
	if userID == actor.ID {
		return validationError{field: "role", msg: "you cannot change your own role — ask another admin"}
	}
	if !validManagedRole(role) {
		return validationError{field: "role", msg: "pick one of admin, approver, analyst or client"}
	}
	if err := a.directory.SetRole(ctx, userID, role, isAgency); err != nil {
		return err
	}
	_ = a.auditor.Record(ctx, actor.ID, actor.Name, "Changed role to "+role, "user", userID)
	return nil
}

// ResetPassword issues a new password and returns it once.
//
// This is the whole of password recovery in this product. A self-serve
// "email me a link" flow needs a mailer that sends, and the only one
// implemented writes the message to the log — a reset link in a log file
// is worse than no reset at all. An admin handing over a generated
// password out of band is the honest version of the same thing.
func (a *Auth) ResetPassword(ctx context.Context, actor identity.User, userID string) (string, error) {
	if !actor.CanManageUsers() {
		return "", ErrNotPermitted
	}
	password, err := generatePassword()
	if err != nil {
		return "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	if err := a.store.SetPassword(ctx, userID, hash); err != nil {
		return "", err
	}
	// Recorded against the admin who did it, because a password changing
	// hands is exactly the event an audit trail exists for.
	_ = a.auditor.Record(ctx, actor.ID, actor.Name, "Reset password", "user", userID)
	return password, nil
}

// generatePassword returns 24 bytes of entropy, URL-safe so it survives a
// terminal, a password manager and a browser form. Matches cmd/createuser.
func generatePassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
