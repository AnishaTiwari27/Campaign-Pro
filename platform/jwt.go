package platform

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the access token's payload — just enough for the gateway to
// answer "who is this, and what are they allowed to do" without a database
// round trip. Role is a coarse "viewer" | "admin" gate (the only two roles
// auth-service issues; see db/init/03_auth.sql).
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// SignAccessToken issues an HS256 JWT for userID/email/role, valid for ttl.
// Only auth-service calls this — it's the one place in the system that
// mints tokens.
func SignAccessToken(secret []byte, userID, email, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "campaign-tracker-pro",
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ParseAccessToken verifies tokenString's signature and expiry against
// secret and returns its claims. The gateway calls this on every protected
// request; no other service parses a JWT — see docs/ARCHITECTURE.md's
// trust-boundary note.
func ParseAccessToken(secret []byte, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
