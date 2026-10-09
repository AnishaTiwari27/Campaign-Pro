package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode"

	"campaigntrackerpro/services/identity"
)

// ErrEmailTaken is returned when the address already has an account.
//
// Unlike a failed sign-in, this does tell the caller that an email is
// registered — and it has to. A signup form that accepted a duplicate
// silently would leave the person with no account and no explanation,
// and one that reported a generic failure would send them to support.
// The disclosure is the price of a usable form; the sign-in path, where
// there is no such excuse, stays deliberately vague.
var ErrEmailTaken = errors.New("an account with that email already exists")

// SignupRole is what a self-registered account gets. Analyst reads
// everything in scope and decides nothing: no approvals, no user
// management, no settings. Anyone who can reach the page can therefore
// look, and nothing more. Promoting someone is a deliberate act by an
// admin, never a side effect of signing up.
const SignupRole = identity.RoleAnalyst

const (
	// MinPasswordLength is a floor, not a policy. Composition rules push
	// people towards "Password1!" — length is what actually helps.
	MinPasswordLength = 10
	// MaxPasswordLength is bcrypt's own limit: it refuses an input longer
	// than 72 bytes outright. Checking here turns what would be a 500 into
	// a sentence the form can show.
	MaxPasswordLength = 72
	// MaxNameLength keeps a display name from overflowing every table and
	// audit line it appears in.
	MaxNameLength = 80
)

// Directory is the capability signup needs: campaigns owns the users
// table, so identity asks it to write rather than writing a table that is
// not its own. Expressed in primitives, exactly as Auditor is, so identity
// stays free of another service's domain types.
type Directory interface {
	// EmailTaken reports whether an account already exists for the address.
	EmailTaken(ctx context.Context, email string) (bool, error)
	// CreateUser writes the account and returns its ID.
	CreateUser(ctx context.Context, email, name, role string, isAgency bool) (string, error)
	// ListUsers is the roster, for the screen that manages it.
	ListUsers(ctx context.Context) ([]identity.DirectoryUser, error)
	// SetRole changes what an account may do.
	SetRole(ctx context.Context, userID, role string, isAgency bool) error
}

// SignupInput is what the form collects, before validation.
type SignupInput struct {
	Name     string
	Email    string
	Password string
}

// Signup registers an account and does *not* sign it in. The new user is
// sent to the sign-in form to use the password they just chose, which
// proves they can before they are relying on the session. It also keeps
// this endpoint from being a way to mint sessions.
func (a *Auth) Signup(ctx context.Context, in SignupInput) (identity.User, error) {
	clean, err := validateSignup(in)
	if err != nil {
		return identity.User{}, err
	}
	if a.directory == nil {
		return identity.User{}, errors.New("signup is not configured on this server")
	}

	// A check, not a guarantee: two requests can pass it at once. The
	// unique index on users.email is the actual guard, and a violation
	// surfaces from CreateUser below. This exists so the ordinary case
	// gets the clear message rather than a constraint error.
	taken, err := a.directory.EmailTaken(ctx, clean.Email)
	if err != nil {
		return identity.User{}, fmt.Errorf("check email: %w", err)
	}
	if taken {
		return identity.User{}, ErrEmailTaken
	}

	// Hash before writing the row. If hashing fails, nothing has been
	// created, so there is no half-made account with no way to sign in.
	hash, err := HashPassword(clean.Password)
	if err != nil {
		return identity.User{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := a.directory.CreateUser(ctx, clean.Email, clean.Name, string(SignupRole), true)
	if err != nil {
		return identity.User{}, fmt.Errorf("create user: %w", err)
	}
	if err := a.store.SetPassword(ctx, id, hash); err != nil {
		return identity.User{}, fmt.Errorf("set password: %w", err)
	}

	user := signupUser(id, clean.Email, clean.Name)
	// Self-registration is security-relevant: it is the one way an account
	// appears without an admin doing it, so the audit trail must show it.
	_ = a.auditor.Record(ctx, user.ID, user.Name, "Signed up", "user", user.ID)
	return user, nil
}

// validateSignup normalises and checks the form. Separate from Signup so
// the rules can be tested without a database.
func validateSignup(in SignupInput) (SignupInput, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return SignupInput{}, validationError{field: "name", msg: "your name is required"}
	}
	if len([]rune(name)) > MaxNameLength {
		return SignupInput{}, validationError{
			field: "name",
			msg:   fmt.Sprintf("name must be %d characters or fewer", MaxNameLength),
		}
	}

	// Lowercased because sign-in lowercases too; without this, signing up
	// as Sam@example.com and signing in as sam@example.com would disagree.
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return SignupInput{}, validationError{field: "email", msg: "an email address is required"}
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		// ParseAddress accepts `Name <a@b.com>`; the address must be bare,
		// or the stored email would not match what is typed at sign-in.
		return SignupInput{}, validationError{field: "email", msg: "that does not look like an email address"}
	}

	// Not trimmed: leading and trailing spaces are legitimate password
	// characters, and silently removing them would lock people out of an
	// account they can no longer reproduce the password for.
	if n := len(in.Password); n < MinPasswordLength {
		return SignupInput{}, validationError{
			field: "password",
			msg:   fmt.Sprintf("password must be at least %d characters", MinPasswordLength),
		}
	} else if n > MaxPasswordLength {
		return SignupInput{}, validationError{
			field: "password",
			msg:   fmt.Sprintf("password must be %d characters or fewer", MaxPasswordLength),
		}
	}
	if strings.TrimFunc(in.Password, unicode.IsSpace) == "" {
		return SignupInput{}, validationError{field: "password", msg: "password cannot be only spaces"}
	}

	return SignupInput{Name: name, Email: email, Password: in.Password}, nil
}

// validationError carries which field was wrong, so the API can point the
// form at it instead of showing one message above everything.
type validationError struct {
	field string
	msg   string
}

func (e validationError) Error() string { return e.msg }
func (e validationError) Field() string { return e.field }

// AsValidationError reports whether err came from signup validation, and
// if so yields the message and the field it belongs to.
func AsValidationError(err error) (msg, field string, ok bool) {
	var v validationError
	if errors.As(err, &v) {
		return v.msg, v.field, true
	}
	return "", "", false
}

// signupUser is the account Signup produces, in one place so a test can
// assert what a stranger is actually granted.
func signupUser(id, email, name string) identity.User {
	return identity.User{ID: id, Email: email, Name: name, Role: SignupRole, IsAgency: true}
}
