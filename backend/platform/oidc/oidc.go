// Package oidc implements the OpenID Connect authorization-code flow
// against any compliant provider.
//
// Standard library only, deliberately. The usual choice here is
// coreos/go-oidc, and it is a good library; this environment cannot
// reliably reach the module proxy, and adding a dependency that cannot
// be fetched means code that cannot be built or tested locally. The
// protocol is well specified and the parts that matter — signature
// verification and claim validation — are better owned explicitly than
// assumed, so they are written out here with tests against a provider
// that signs real tokens.
//
// What is enforced, because each of these is a real attack if skipped:
//
//	state     CSRF. A callback without the state we issued is refused.
//	nonce     Replay. Bound into the ID token and compared on return.
//	PKCE      Code interception. S256 only; "plain" is not offered.
//	signature RS256 against the provider's JWKS, matched by kid.
//	iss/aud   The token was issued by who we asked, for us.
//	exp/iat   It is current, within a small clock skew.
package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// clockSkew is how far out of step a provider's clock may be before a
// token is rejected. Thirty seconds is the usual allowance; larger
// widens the replay window for an expired token.
const clockSkew = 30 * time.Second

// Config is a provider plus our registration with it.
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	// HTTPClient is injectable so tests can point at a local provider.
	HTTPClient *http.Client
}

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// discovery is the provider's advertised configuration.
type discovery struct {
	Issuer      string `json:"issuer"`
	AuthURL     string `json:"authorization_endpoint"`
	TokenURL    string `json:"token_endpoint"`
	JWKSURL     string `json:"jwks_uri"`
	UserInfoURL string `json:"userinfo_endpoint"`
}

// Provider is a configured, discovered identity provider.
type Provider struct {
	cfg  Config
	disc discovery

	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// Discover reads the provider's well-known configuration.
//
// Done once at startup rather than per request: it is the slowest part
// of the flow, and a provider that is unreachable at boot is a
// misconfiguration worth failing loudly for rather than discovering on
// somebody's first sign-in.
func Discover(ctx context.Context, cfg Config) (*Provider, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("oidc: no issuer configured")
	}
	wellKnown := strings.TrimSuffix(cfg.Issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, err
	}
	resp, err := cfg.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: cannot reach %s: %w", wellKnown, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc: discovery returned %s", resp.Status)
	}
	var d discovery
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&d); err != nil {
		return nil, fmt.Errorf("oidc: malformed discovery document: %w", err)
	}
	// The issuer in the document must match the one configured.
	// Otherwise a provider could claim to be someone else and every
	// later iss check would be comparing against the wrong name.
	if d.Issuer != strings.TrimSuffix(cfg.Issuer, "/") && d.Issuer != cfg.Issuer {
		return nil, fmt.Errorf("oidc: discovery issuer %q does not match configured issuer %q", d.Issuer, cfg.Issuer)
	}
	if d.AuthURL == "" || d.TokenURL == "" || d.JWKSURL == "" {
		return nil, fmt.Errorf("oidc: discovery document is missing required endpoints")
	}
	return &Provider{cfg: cfg, disc: d, keys: map[string]*rsa.PublicKey{}}, nil
}

// AuthRequest is the per-sign-in state that must survive the round trip
// to the provider and be checked on the way back.
type AuthRequest struct {
	URL      string
	State    string
	Nonce    string
	Verifier string
}

// Start builds the redirect to the provider.
func (p *Provider) Start() (AuthRequest, error) {
	state, err := randomString()
	if err != nil {
		return AuthRequest{}, err
	}
	nonce, err := randomString()
	if err != nil {
		return AuthRequest{}, err
	}
	verifier, err := randomString()
	if err != nil {
		return AuthRequest{}, err
	}
	challenge := s256(verifier)

	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.cfg.ClientID},
		"redirect_uri":          {p.cfg.RedirectURL},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	sep := "?"
	if strings.Contains(p.disc.AuthURL, "?") {
		sep = "&"
	}
	return AuthRequest{
		URL:      p.disc.AuthURL + sep + q.Encode(),
		State:    state,
		Nonce:    nonce,
		Verifier: verifier,
	}, nil
}

// Identity is who the provider says signed in.
type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

type tokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// Exchange trades the authorization code for an ID token and validates
// it. The nonce is the one issued at Start; a token carrying any other
// is a replay and is refused.
func (p *Provider) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURL},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.disc.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.cfg.client().Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("oidc: token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tr); err != nil {
		return Identity{}, fmt.Errorf("oidc: malformed token response: %w", err)
	}
	if tr.Error != "" {
		return Identity{}, fmt.Errorf("oidc: provider refused the exchange: %s %s", tr.Error, tr.ErrorDesc)
	}
	if tr.IDToken == "" {
		return Identity{}, fmt.Errorf("oidc: token response carried no id_token")
	}
	return p.verify(ctx, tr.IDToken, nonce)
}

type idClaims struct {
	Issuer        string          `json:"iss"`
	Subject       string          `json:"sub"`
	Audience      json.RawMessage `json:"aud"`
	Expiry        int64           `json:"exp"`
	IssuedAt      int64           `json:"iat"`
	Nonce         string          `json:"nonce"`
	Email         string          `json:"email"`
	EmailVerified any             `json:"email_verified"`
	Name          string          `json:"name"`
}

// verify checks the signature first and the claims second. In that
// order deliberately: claims from an unverified token are attacker-
// controlled strings, and reading them before checking the signature
// invites decisions based on them.
func (p *Provider) verify(ctx context.Context, raw, nonce string) (Identity, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Identity{}, fmt.Errorf("oidc: id_token is not a JWT")
	}
	headerJSON, err := b64(parts[0])
	if err != nil {
		return Identity{}, fmt.Errorf("oidc: bad token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return Identity{}, fmt.Errorf("oidc: bad token header: %w", err)
	}
	// RS256 only. "none" is the textbook forgery, and allowing HS256
	// alongside RS256 is the other one: the public key then doubles as
	// an HMAC secret anyone can read.
	if header.Alg != "RS256" {
		return Identity{}, fmt.Errorf("oidc: unsupported signing algorithm %q", header.Alg)
	}

	key, err := p.keyFor(ctx, header.Kid)
	if err != nil {
		return Identity{}, err
	}
	sig, err := b64(parts[2])
	if err != nil {
		return Identity{}, fmt.Errorf("oidc: bad signature encoding: %w", err)
	}
	signed := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, signed[:], sig); err != nil {
		return Identity{}, fmt.Errorf("oidc: signature does not verify")
	}

	payload, err := b64(parts[1])
	if err != nil {
		return Identity{}, fmt.Errorf("oidc: bad token payload: %w", err)
	}
	var c idClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Identity{}, fmt.Errorf("oidc: bad token payload: %w", err)
	}

	if c.Issuer != p.disc.Issuer {
		return Identity{}, fmt.Errorf("oidc: token issued by %q, expected %q", c.Issuer, p.disc.Issuer)
	}
	if !audienceContains(c.Audience, p.cfg.ClientID) {
		return Identity{}, fmt.Errorf("oidc: token is not addressed to this client")
	}
	now := time.Now()
	if c.Expiry == 0 || now.After(time.Unix(c.Expiry, 0).Add(clockSkew)) {
		return Identity{}, fmt.Errorf("oidc: token has expired")
	}
	if c.IssuedAt != 0 && now.Add(clockSkew).Before(time.Unix(c.IssuedAt, 0)) {
		return Identity{}, fmt.Errorf("oidc: token was issued in the future")
	}
	// Compared even when empty: a provider that drops the nonce has
	// removed the replay protection, and accepting that silently
	// defeats the purpose of sending one.
	if c.Nonce != nonce {
		return Identity{}, fmt.Errorf("oidc: nonce does not match — possible replay")
	}
	if c.Subject == "" {
		return Identity{}, fmt.Errorf("oidc: token carries no subject")
	}

	return Identity{
		Subject:       c.Subject,
		Email:         strings.ToLower(strings.TrimSpace(c.Email)),
		EmailVerified: truthy(c.EmailVerified),
		Name:          c.Name,
	}, nil
}

// audienceContains handles aud being either a string or an array, which
// the spec permits and providers differ on.
func audienceContains(raw json.RawMessage, want string) bool {
	if len(raw) == 0 {
		return false
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return one == want
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		for _, a := range many {
			if a == want {
				return true
			}
		}
	}
	return false
}

// truthy handles email_verified arriving as a bool or as a string,
// which some providers do.
func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true"
	default:
		return false
	}
}

// keyFor returns the signing key with this id, refetching the key set
// if it is unknown. Providers rotate keys without warning, so an
// unrecognised kid means "fetch again", not "reject" — but the refetch
// is rate-limited so a bogus kid cannot be used to hammer the provider.
func (p *Provider) keyFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	p.mu.RLock()
	k, ok := p.keys[kid]
	age := time.Since(p.fetched)
	p.mu.RUnlock()
	if ok {
		return k, nil
	}
	if age < time.Minute {
		return nil, fmt.Errorf("oidc: no signing key %q in the provider's key set", kid)
	}
	if err := p.refreshKeys(ctx); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if k, ok := p.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("oidc: no signing key %q in the provider's key set", kid)
}

type jwks struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (p *Provider) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.disc.JWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.cfg.client().Do(req)
	if err != nil {
		return fmt.Errorf("oidc: cannot fetch key set: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc: key set returned %s", resp.Status)
	}
	var set jwks
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return fmt.Errorf("oidc: malformed key set: %w", err)
	}

	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		nb, err := b64(k.N)
		if err != nil {
			continue
		}
		eb, err := b64(k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(bigEndian(eb))}
	}
	if len(keys) == 0 {
		return fmt.Errorf("oidc: key set contained no usable RSA signing keys")
	}
	p.mu.Lock()
	p.keys, p.fetched = keys, time.Now()
	p.mu.Unlock()
	return nil
}

// bigEndian reads a JWK exponent, which is a big-endian byte string of
// unpredictable length — commonly three bytes for 65537.
func bigEndian(b []byte) uint64 {
	var padded [8]byte
	if len(b) > 8 {
		b = b[len(b)-8:]
	}
	copy(padded[8-len(b):], b)
	return binary.BigEndian.Uint64(padded[:])
}

func b64(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

func s256(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// randomString is 256 bits from crypto/rand. Used for state, nonce and
// the PKCE verifier, all of which are only as good as their entropy.
func randomString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
