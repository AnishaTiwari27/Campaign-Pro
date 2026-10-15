// Package oauth refreshes access tokens for the ad platforms.
//
// Both Google and Meta hand out a long-lived refresh token once and
// expect short-lived access tokens to be minted from it. Doing that
// per request would be slow and would hammer the token endpoint, and
// doing it once at boot would break after an hour, so the token is
// cached and renewed slightly before it expires.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// earlyRefresh renews a token before it actually expires. A token that
// expires mid-flight produces a 401 on a sync that was otherwise fine,
// and a minute of slack costs nothing.
const earlyRefresh = 60 * time.Second

// RefreshTokenSource mints access tokens from a refresh token.
type RefreshTokenSource struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	RefreshToken string
	HTTP         *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func (s *RefreshTokenSource) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Token returns a valid access token, refreshing if the cached one is
// spent. Safe for concurrent use: two syncs starting together must not
// both refresh, and must not hand each other a half-written token.
func (s *RefreshTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Add(earlyRefresh).Before(s.expiresAt) {
		return s.token, nil
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {s.RefreshToken},
		"client_id":     {s.ClientID},
		"client_secret": {s.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth: refresh failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var tr tokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return "", fmt.Errorf("oauth: malformed token response: %w", err)
	}
	if tr.Error != "" {
		// Surfaced as-is: "invalid_grant" means the refresh token has
		// been revoked and someone has to re-authorise, which is a
		// different job from retrying.
		return "", fmt.Errorf("oauth: %s: %s", tr.Error, tr.ErrorDesc)
	}
	if resp.StatusCode != http.StatusOK || tr.AccessToken == "" {
		return "", fmt.Errorf("oauth: token endpoint returned %s", resp.Status)
	}

	ttl := time.Duration(tr.ExpiresIn) * time.Second
	if ttl <= 0 {
		// Google omits expires_in in some responses. An hour is the
		// documented default and the floor is what matters: too short
		// costs an extra refresh, too long costs a failed sync.
		ttl = time.Hour
	}
	s.token, s.expiresAt = tr.AccessToken, time.Now().Add(ttl)
	return s.token, nil
}
