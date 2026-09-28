package platform

import (
	"testing"
	"time"
)

func TestSignAndParseAccessToken_RoundTrip(t *testing.T) {
	secret := []byte("test-secret")
	tok, err := SignAccessToken(secret, "user-1", "a@b.com", "admin", time.Minute)
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	claims, err := ParseAccessToken(secret, tok)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "a@b.com" || claims.Role != "admin" {
		t.Fatalf("claims = %+v, want UserID=user-1 Email=a@b.com Role=admin", claims)
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	secret := []byte("test-secret")
	tok, err := SignAccessToken(secret, "user-1", "a@b.com", "viewer", -time.Minute) // already expired
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if _, err := ParseAccessToken(secret, tok); err == nil {
		t.Fatal("expected an error parsing an expired token, got nil")
	}
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	tok, err := SignAccessToken([]byte("secret-a"), "user-1", "a@b.com", "viewer", time.Minute)
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if _, err := ParseAccessToken([]byte("secret-b"), tok); err == nil {
		t.Fatal("expected a signature error parsing with the wrong secret, got nil")
	}
}

func TestParseAccessToken_Garbage(t *testing.T) {
	if _, err := ParseAccessToken([]byte("test-secret"), "not-a-jwt"); err == nil {
		t.Fatal("expected an error parsing a non-JWT string, got nil")
	}
}
