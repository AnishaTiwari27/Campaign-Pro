// Command createuser creates (or re-passwords) one account, without touching
// anything else.
//
// It exists because `make seed` begins with TRUNCATE: it is a fixture loader,
// and running it against a real deployment would erase that deployment. There
// is no user-management endpoint yet either, so this is the supported way to
// make the first real admin — and, since the app has no password-reset flow,
// the way to recover an account whose password has been lost.
//
// Point it at the deployment's database from your own machine:
//
//	DATABASE_URL='postgres://...' go run ./cmd/createuser -email you@example.com -name 'Your Name'
//
// With no -password it generates a strong one and prints it once. Passing a
// password on the command line puts it in your shell history and in the
// process list on a shared host, so prefer the generated one.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"campaigntrackerpro/platform/config"
	"campaigntrackerpro/platform/database"
	campaignsmod "campaigntrackerpro/services/campaigns/module"
	"campaigntrackerpro/services/identity"
	identitymod "campaigntrackerpro/services/identity/module"
)

func main() {
	email := flag.String("email", "", "email address to sign in with (required)")
	name := flag.String("name", "", "display name shown in the UI and the audit trail (required)")
	role := flag.String("role", string(identity.RoleAdmin), "admin | approver | analyst | client")
	password := flag.String("password", "", "password; generated and printed if omitted (preferred)")
	agency := flag.Bool("agency", true, "agency staff, who see every account; false for an external client")
	flag.Parse()

	if *email == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "both -email and -name are required")
		flag.Usage()
		os.Exit(2)
	}
	if err := validRole(*role); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	plain := *password
	generated := plain == ""
	if generated {
		var err error
		if plain, err = generatePassword(); err != nil {
			fmt.Fprintf(os.Stderr, "could not generate a password: %v\n", err)
			os.Exit(1)
		}
	}

	ctx := context.Background()
	cfg := config.Load()
	// Nothing this command does is worth logging; errors are reported to the
	// operator on stderr instead.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := database.New(pool)

	camp := campaignsmod.New(db, logger)
	ident := identitymod.New(db, noopAuditor{}, false, logger)

	// can_approve mirrors the role, so the column and identity's CanApprove()
	// cannot disagree about the same account.
	canApprove := *role == string(identity.RoleAdmin) || *role == string(identity.RoleApprover)

	existing, err := camp.Store().GetUserByEmail(ctx, *email)
	switch {
	case err == nil:
		// Re-password rather than fail: an operator who has lost the admin
		// password has no other way back in, and refusing here would leave
		// them editing the users table by hand.
		if perr := ident.SetPassword(ctx, existing.ID, plain); perr != nil {
			fmt.Fprintf(os.Stderr, "set password: %v\n", perr)
			os.Exit(1)
		}
		fmt.Printf("updated the password for the existing account %s (role %s, unchanged)\n", *email, existing.Role)
		report(*email, plain, generated)
		return
	case !errors.Is(err, database.ErrNotFound):
		fmt.Fprintf(os.Stderr, "look up %s: %v\n", *email, err)
		os.Exit(1)
	}

	created, err := camp.Store().CreateUser(ctx, *email, *name, *role, canApprove)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		os.Exit(1)
	}
	if err := camp.Store().SetUserAgency(ctx, created.ID, *agency); err != nil {
		fmt.Fprintf(os.Stderr, "set agency flag: %v\n", err)
		os.Exit(1)
	}
	if err := ident.SetPassword(ctx, created.ID, plain); err != nil {
		fmt.Fprintf(os.Stderr, "set password: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created %s as %s (agency=%t, can approve=%t)\n", *email, *role, *agency, canApprove)
	report(*email, plain, generated)
}

func report(email, plain string, generated bool) {
	if !generated {
		fmt.Println("password: the one you supplied")
		return
	}
	fmt.Printf("\n  email:    %s\n  password: %s\n\n", email, plain)
	fmt.Println("This password is shown once and is not stored anywhere in readable form.")
	fmt.Println("Save it now, then sign in and treat it as the account's real credential.")
}

// generatePassword returns 24 bytes of entropy, URL-safe so it survives being
// copied through a terminal, a password manager and a browser form.
func generatePassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func validRole(role string) error {
	for _, r := range []identity.Role{
		identity.RoleAdmin, identity.RoleApprover, identity.RoleAnalyst, identity.RoleClient,
	} {
		if role == string(r) {
			return nil
		}
	}
	return fmt.Errorf("unknown role %q: use admin, approver, analyst or client", role)
}

// noopAuditor satisfies identity's auditor dependency. A password written by
// an operator at a terminal is not a sign-in event.
type noopAuditor struct{}

func (noopAuditor) Record(ctx context.Context, userID, actor, action, entityType, entityID string) error {
	return nil
}
