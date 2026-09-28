package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// newOpaqueToken returns a random, URL-safe refresh token. It's returned to
// the client as-is and never stored — only its hash is (see hashToken and
// db/init/03_auth.sql) — so a database leak alone can't be replayed as a
// live session.
func newOpaqueToken() (string, error) {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken is the lookup key stored in auth.refresh_tokens — sha256 is
// enough here (unlike a password hash, an opaque 256-bit token has no
// guessable structure for a fast hash to matter).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
