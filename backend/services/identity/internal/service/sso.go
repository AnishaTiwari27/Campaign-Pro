package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"campaigntrackerpro/platform/database"
	"campaigntrackerpro/platform/oidc"
	"campaigntrackerpro/platform/scope"
	"campaigntrackerpro/services/identity"
)

// ErrSSOUnverifiedEmail is returned when the provider has not verified
// the address it is asserting.
//
// The email is the only thing linking an SSO identity to an existing
// account, so an unverified one would let anyone who can set their
// profile address at the provider sign in as somebody else here.
var ErrSSOUnverifiedEmail = errors.New("the identity provider has not verified this email address")

// SSOLogin signs a user in from a verified provider identity.
//
// Just-in-time provisioning: an identity the provider vouches for, that
// we have never seen, becomes an account. The alternative is an admin
// creating every row by hand before anyone can use SSO, which defeats
// most of the point of having it.
//
// The new account is an analyst, the least privileged role that can
// actually use the product. Roles are the agency's to grant and a
// provider has no idea what they mean here, so nobody arrives able to
// approve spend.
func (a *Auth) SSOLogin(ctx context.Context, id oidc.Identity, orgID, userAgent, ip string) (identity.User, string, error) {
	if id.Email == "" {
		return identity.User{}, "", fmt.Errorf("the identity provider returned no email address")
	}
	if !id.EmailVerified {
		return identity.User{}, "", ErrSSOUnverifiedEmail
	}
	email := strings.TrimSpace(strings.ToLower(id.Email))

	// Resolution happens inside the organisation the issuer maps to, so
	// the same address at two different providers cannot collide.
	orgCtx := scope.WithOrg(ctx, scope.Org{ID: orgID})

	user, _, err := a.store.UserForAuth(orgCtx, email)
	switch {
	case err == nil:
		// Existing account.
	case errors.Is(err, database.ErrNotFound):
		if a.directory == nil {
			return identity.User{}, "", fmt.Errorf("no account for %s and provisioning is not configured", email)
		}
		name := id.Name
		if name == "" {
			name = email
		}
		// isAgency: an account arriving through the agency's own
		// identity provider is agency staff. A client is granted
		// specific campaigns by hand and does not come through here.
		if _, err := a.directory.CreateUser(orgCtx, email, name, string(identity.RoleAnalyst), true); err != nil {
			return identity.User{}, "", fmt.Errorf("provision %s: %w", email, err)
		}
		user, _, err = a.store.UserForAuth(orgCtx, email)
		if err != nil {
			return identity.User{}, "", err
		}
		_ = a.auditor.Record(orgCtx, user.ID, user.Name, "Account provisioned via SSO", "user", user.ID)
	default:
		return identity.User{}, "", err
	}

	// No password is checked and none is set. An SSO account has no
	// local credential to steal, and giving it one would reintroduce
	// the thing SSO exists to remove.
	token, err := newToken()
	if err != nil {
		return identity.User{}, "", err
	}
	if err := a.store.CreateSession(orgCtx, token, user.ID, userAgent, ip); err != nil {
		return identity.User{}, "", err
	}
	_ = a.store.TouchLogin(orgCtx, user.ID)
	_ = a.auditor.Record(orgCtx, user.ID, user.Name, "Signed in via SSO", "session", user.ID)

	return user, token, nil
}
