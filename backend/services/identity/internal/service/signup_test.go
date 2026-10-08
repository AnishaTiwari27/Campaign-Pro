package service

import (
	"strings"
	"testing"
)

func TestValidateSignup(t *testing.T) {
	ok := SignupInput{Name: "Asha Rao", Email: "asha@example.com", Password: "correct horse battery"}

	t.Run("a good input is normalised, not rejected", func(t *testing.T) {
		got, err := validateSignup(SignupInput{
			Name: "  Asha Rao  ", Email: "  ASHA@Example.COM ", Password: ok.Password,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "Asha Rao" {
			t.Errorf("name = %q, want %q", got.Name, "Asha Rao")
		}
		// Lowercased to match Login, which lowercases before lookup.
		if got.Email != "asha@example.com" {
			t.Errorf("email = %q, want %q", got.Email, "asha@example.com")
		}
		if got.Password != ok.Password {
			t.Errorf("password was altered: %q", got.Password)
		}
	})

	t.Run("surrounding spaces in a password are preserved", func(t *testing.T) {
		pw := "  spaces count  "
		got, err := validateSignup(SignupInput{Name: ok.Name, Email: ok.Email, Password: pw})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Password != pw {
			t.Fatalf("password = %q, want it untouched %q", got.Password, pw)
		}
	})

	cases := []struct {
		name  string
		in    SignupInput
		field string
	}{
		{"empty name", SignupInput{Name: " ", Email: ok.Email, Password: ok.Password}, "name"},
		{
			"over-long name",
			SignupInput{Name: strings.Repeat("a", MaxNameLength+1), Email: ok.Email, Password: ok.Password},
			"name",
		},
		{"empty email", SignupInput{Name: ok.Name, Email: "", Password: ok.Password}, "email"},
		{"malformed email", SignupInput{Name: ok.Name, Email: "not-an-email", Password: ok.Password}, "email"},
		{
			// mail.ParseAddress accepts this form; the stored address must
			// be bare or sign-in could never match it.
			"display-name form is rejected",
			SignupInput{Name: ok.Name, Email: "asha <asha@example.com>", Password: ok.Password},
			"email",
		},
		{
			"short password",
			SignupInput{Name: ok.Name, Email: ok.Email, Password: strings.Repeat("x", MinPasswordLength-1)},
			"password",
		},
		{
			// bcrypt refuses more than 72 bytes, so this would be a 500
			// rather than a message if it reached the hasher.
			"password past bcrypt's limit",
			SignupInput{Name: ok.Name, Email: ok.Email, Password: strings.Repeat("x", MaxPasswordLength+1)},
			"password",
		},
		{
			"whitespace-only password",
			SignupInput{Name: ok.Name, Email: ok.Email, Password: strings.Repeat(" ", MinPasswordLength+2)},
			"password",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := validateSignup(c.in)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			msg, field, ok := AsValidationError(err)
			if !ok {
				t.Fatalf("error is not a validation error: %v", err)
			}
			if field != c.field {
				t.Errorf("field = %q, want %q", field, c.field)
			}
			if msg == "" {
				t.Error("validation error carried no message for the form to show")
			}
		})
	}
}

// The role a stranger gets is the whole security question for this feature,
// so it is pinned rather than left to whatever the constant happens to say.
func TestSignupRoleGrantsNoAuthority(t *testing.T) {
	u := signupUser("id", "a@b.com", "A")
	if u.CanApprove() {
		t.Error("a self-registered account can approve — it must not")
	}
	if u.CanManageUsers() {
		t.Error("a self-registered account can manage users — it must not")
	}
	if u.IsClient() {
		t.Error("a self-registered account reads as an external client, so agency data would be suppressed")
	}
}
