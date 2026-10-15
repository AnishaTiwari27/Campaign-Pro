package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeIdP is a provider that signs real RS256 tokens, so these tests
// exercise the actual verification path rather than a stub of it.
type fakeIdP struct {
	*httptest.Server
	key      *rsa.PrivateKey
	kid      string
	clientID string

	// knobs the tests turn to forge a token
	issuerOverride string
	audOverride    any
	expOverride    int64
	algOverride    string
	nonceOverride  *string
	signWithOther  bool
	otherKey       *rsa.PrivateKey
}

func newIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	idp := &fakeIdP{key: key, otherKey: other, kid: "test-key-1", clientID: "test-client"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 idp.URL,
			"authorization_endpoint": idp.URL + "/authorize",
			"token_endpoint":         idp.URL + "/token",
			"jwks_uri":               idp.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}) // 65537
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{
			{"kid": idp.kid, "kty": "RSA", "alg": "RS256", "use": "sig", "n": n, "e": e},
		}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		nonce := "test-nonce"
		if idp.nonceOverride != nil {
			nonce = *idp.nonceOverride
		}
		json.NewEncoder(w).Encode(map[string]string{
			"id_token": idp.signToken(t, nonce), "token_type": "Bearer",
		})
	})
	idp.Server = httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}

func (f *fakeIdP) signToken(t *testing.T, nonce string) string {
	t.Helper()
	alg := "RS256"
	if f.algOverride != "" {
		alg = f.algOverride
	}
	iss := f.URL
	if f.issuerOverride != "" {
		iss = f.issuerOverride
	}
	var aud any = f.clientID
	if f.audOverride != nil {
		aud = f.audOverride
	}
	exp := time.Now().Add(time.Hour).Unix()
	if f.expOverride != 0 {
		exp = f.expOverride
	}

	header, _ := json.Marshal(map[string]string{"alg": alg, "kid": f.kid, "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss": iss, "sub": "user-42", "aud": aud,
		"exp": exp, "iat": time.Now().Unix(), "nonce": nonce,
		"email": "sso.user@example.test", "email_verified": true, "name": "SSO User",
	})
	signing := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)

	key := f.key
	if f.signWithOther {
		key = f.otherKey
	}
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func discover(t *testing.T, idp *fakeIdP) *Provider {
	t.Helper()
	p, err := Discover(context.Background(), Config{
		Issuer: idp.URL, ClientID: idp.clientID, ClientSecret: "secret",
		RedirectURL: "http://localhost/callback", HTTPClient: idp.Client(),
	})
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}
	return p
}

func TestTheHappyPathReturnsTheIdentity(t *testing.T) {
	idp := newIdP(t)
	p := discover(t, idp)

	id, err := p.Exchange(context.Background(), "the-code", "the-verifier", "test-nonce")
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}
	if id.Subject != "user-42" || id.Email != "sso.user@example.test" || !id.EmailVerified {
		t.Errorf("identity wrong: %+v", id)
	}
}

// The authorization request has to carry everything the callback will
// be checked against. A missing nonce or a "plain" PKCE challenge is a
// flow that looks like it works and protects nothing.
func TestStartSendsStateNonceAndS256PKCE(t *testing.T) {
	idp := newIdP(t)
	p := discover(t, idp)

	req, err := p.Start()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(req.URL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()

	if q.Get("state") != req.State || req.State == "" {
		t.Error("state is missing or does not match what was returned")
	}
	if q.Get("nonce") != req.Nonce || req.Nonce == "" {
		t.Error("nonce is missing or does not match what was returned")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("challenge method is %q, want S256", q.Get("code_challenge_method"))
	}
	if got, want := q.Get("code_challenge"), s256(req.Verifier); got != want {
		t.Error("the challenge is not the SHA-256 of the verifier")
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type is %q, want code", q.Get("response_type"))
	}
	// Two sign-ins must not share a state, or one tab's callback would
	// satisfy another tab's request.
	second, _ := p.Start()
	if second.State == req.State || second.Nonce == req.Nonce || second.Verifier == req.Verifier {
		t.Error("two authorization requests reused the same random values")
	}
}

// Each of these is a forged or stale token that must be refused. If any
// one of them passes, the flow authenticates an attacker.
func TestForgedAndStaleTokensAreRefused(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(*fakeIdP)
		nonce  string
		expect string
	}{
		{
			name:   "signed by a key the provider does not publish",
			setup:  func(f *fakeIdP) { f.signWithOther = true },
			nonce:  "test-nonce",
			expect: "signature",
		},
		{
			name:   "alg none, the textbook forgery",
			setup:  func(f *fakeIdP) { f.algOverride = "none" },
			nonce:  "test-nonce",
			expect: "unsupported signing algorithm",
		},
		{
			name:   "HS256, which would let the public key act as a shared secret",
			setup:  func(f *fakeIdP) { f.algOverride = "HS256" },
			nonce:  "test-nonce",
			expect: "unsupported signing algorithm",
		},
		{
			name:   "issued by somebody else",
			setup:  func(f *fakeIdP) { f.issuerOverride = "https://evil.example" },
			nonce:  "test-nonce",
			expect: "issued by",
		},
		{
			name:   "addressed to a different client",
			setup:  func(f *fakeIdP) { f.audOverride = "some-other-client" },
			nonce:  "test-nonce",
			expect: "not addressed to this client",
		},
		{
			name:   "expired",
			setup:  func(f *fakeIdP) { f.expOverride = time.Now().Add(-time.Hour).Unix() },
			nonce:  "test-nonce",
			expect: "expired",
		},
		{
			name:   "replayed with a nonce from another sign-in",
			setup:  func(f *fakeIdP) {},
			nonce:  "a-different-nonce",
			expect: "nonce does not match",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idp := newIdP(t)
			p := discover(t, idp)
			tc.setup(idp)

			_, err := p.Exchange(context.Background(), "code", "verifier", tc.nonce)
			if err == nil {
				t.Fatal("the token was accepted — this is an authentication bypass")
			}
			if !strings.Contains(err.Error(), tc.expect) {
				t.Errorf("refused for the wrong reason:\n  got:  %v\n  want: something containing %q", err, tc.expect)
			}
		})
	}
}

// aud may be a string or an array; providers differ and both are legal.
func TestAudienceMayBeAnArray(t *testing.T) {
	idp := newIdP(t)
	p := discover(t, idp)
	idp.audOverride = []string{"someone-else", idp.clientID}

	if _, err := p.Exchange(context.Background(), "code", "verifier", "test-nonce"); err != nil {
		t.Errorf("an array audience containing our client id was refused: %v", err)
	}
}

// A provider claiming an issuer other than the one configured would
// make every later iss check compare against the wrong name.
func TestDiscoveryRejectsAnIssuerMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 "https://not-who-you-asked-for.example",
			"authorization_endpoint": "https://x/authorize",
			"token_endpoint":         "https://x/token",
			"jwks_uri":               "https://x/jwks",
		})
	}))
	defer srv.Close()

	_, err := Discover(context.Background(), Config{
		Issuer: srv.URL, ClientID: "c", ClientSecret: "s",
		RedirectURL: "http://localhost/cb", HTTPClient: srv.Client(),
	})
	if err == nil || !strings.Contains(err.Error(), "does not match configured issuer") {
		t.Fatalf("expected an issuer mismatch refusal, got %v", err)
	}
}

// A provider that returns an OAuth error must surface it rather than
// failing on a missing id_token, which sends people to debug the wrong
// thing.
func TestAProviderErrorIsReportedAsItself(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			base := "http://" + r.Host
			json.NewEncoder(w).Encode(map[string]string{
				"issuer": base, "authorization_endpoint": base + "/a",
				"token_endpoint": base + "/token", "jwks_uri": base + "/jwks",
			})
		case "/token":
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid_grant", "error_description": "code already used",
			})
		}
	}))
	defer srv.Close()

	p, err := Discover(context.Background(), Config{
		Issuer: srv.URL, ClientID: "c", ClientSecret: "s",
		RedirectURL: "http://localhost/cb", HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exchange(context.Background(), "code", "verifier", "n")
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("expected the provider's own error, got %v", err)
	}
}

func TestExponentDecoding(t *testing.T) {
	if got := bigEndian([]byte{0x01, 0x00, 0x01}); got != 65537 {
		t.Errorf("bigEndian(AQAB) = %d, want 65537", got)
	}
	if got := bigEndian([]byte{0x03}); got != 3 {
		t.Errorf("bigEndian(0x03) = %d, want 3", got)
	}
}
