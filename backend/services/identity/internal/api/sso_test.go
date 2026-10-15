package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"campaigntrackerpro/platform/oidc"
)

// stubProvider stands in for a discovered IdP. The protocol itself is
// tested against a real signing provider in platform/oidc; what matters
// here is the HTTP plumbing around it — the state cookie, the CSRF
// check and what the browser is told to do next.
type stubProvider struct {
	req      oidc.AuthRequest
	identity oidc.Identity
	err      error

	gotCode     string
	gotVerifier string
	gotNonce    string
}

func (s *stubProvider) Start() (oidc.AuthRequest, error) { return s.req, nil }

func (s *stubProvider) Exchange(_ context.Context, code, verifier, nonce string) (oidc.Identity, error) {
	s.gotCode, s.gotVerifier, s.gotNonce = code, verifier, nonce
	return s.identity, s.err
}

func testHandlers(p *stubProvider) *Handlers {
	return &Handlers{sso: p, ssoOrgID: "default", logger: discardLogger()}
}

// Start has to put the state somewhere the callback can check it, and
// it has to be out of reach of scripts: the verifier in particular is
// the thing PKCE depends on staying private.
func TestSSOStartSetsAnHttpOnlyStateCookieAndRedirects(t *testing.T) {
	p := &stubProvider{req: oidc.AuthRequest{
		URL: "https://idp.example/authorize?x=1", State: "st", Nonce: "no", Verifier: "ve",
	}}
	h := testHandlers(p)

	rec := httptest.NewRecorder()
	h.SSOStart(rec, httptest.NewRequest(http.MethodGet, "/api/auth/sso/start", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("got %d, want a redirect", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != p.req.URL {
		t.Errorf("redirected to %q, want the provider's authorize URL", loc)
	}

	var c *http.Cookie
	for _, got := range rec.Result().Cookies() {
		if got.Name == ssoStateCookie {
			c = got
		}
	}
	if c == nil {
		t.Fatal("no state cookie was set — the callback will have nothing to check against")
	}
	if !c.HttpOnly {
		t.Error("the state cookie is readable by scripts; it carries the PKCE verifier")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Error("SameSite must be Lax — Strict is dropped on the provider's redirect back and breaks every sign-in")
	}
}

func cookieFor(t *testing.T, st ssoState) *http.Cookie {
	t.Helper()
	blob, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: ssoStateCookie, Value: base64.RawURLEncoding.EncodeToString(blob)}
}

// The CSRF check. Without it an attacker hands someone a callback URL
// carrying the attacker's own code, and the victim is signed into the
// attacker's account — where anything they then do is visible to them.
func TestCallbackRefusesAStateThatDoesNotMatch(t *testing.T) {
	p := &stubProvider{}
	h := testHandlers(p)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?code=c&state=attacker-state", nil)
	r.AddCookie(cookieFor(t, ssoState{State: "our-state", Nonce: "n", Verifier: "v"}))
	rec := httptest.NewRecorder()
	h.SSOCallback(rec, r)

	if p.gotCode != "" {
		t.Error("the code was exchanged despite a state mismatch")
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "sso=failed") {
		t.Errorf("sent to %q, expected the sign-in screen with a failure", loc)
	}
}

// A callback with no cookie at all is an abandoned or forged attempt.
func TestCallbackWithNoStateCookieIsRefused(t *testing.T) {
	p := &stubProvider{}
	h := testHandlers(p)

	rec := httptest.NewRecorder()
	h.SSOCallback(rec, httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?code=c&state=s", nil))

	if p.gotCode != "" {
		t.Error("the code was exchanged with no state to check it against")
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "sso=expired") {
		t.Errorf("sent to %q, expected sso=expired", loc)
	}
}

// The state cookie must be single-use, cleared before anything can
// fail, so a rejected callback cannot be retried against the same nonce.
func TestCallbackClearsTheStateCookieEvenOnFailure(t *testing.T) {
	h := testHandlers(&stubProvider{})
	r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?error=access_denied", nil)
	r.AddCookie(cookieFor(t, ssoState{State: "s", Nonce: "n", Verifier: "v"}))
	rec := httptest.NewRecorder()
	h.SSOCallback(rec, r)

	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == ssoStateCookie && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("the state cookie survived a failed callback and can be replayed")
	}
}

// A user who cancels at the provider is not an error worth a stack
// trace — they go back to the sign-in screen.
func TestAProviderDenialSendsTheUserBackToSignIn(t *testing.T) {
	h := testHandlers(&stubProvider{})
	r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?error=access_denied", nil)
	r.AddCookie(cookieFor(t, ssoState{State: "s", Nonce: "n", Verifier: "v"}))
	rec := httptest.NewRecorder()
	h.SSOCallback(rec, r)

	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "sso=denied") {
		t.Errorf("sent to %q, expected sso=denied", loc)
	}
}

// The verifier and nonce from the cookie have to reach the exchange, or
// PKCE and replay protection are decorative.
func TestCallbackPassesTheStoredVerifierAndNonceToTheExchange(t *testing.T) {
	p := &stubProvider{err: context.Canceled} // stop before the database
	h := testHandlers(p)

	r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/callback?code=the-code&state=the-state", nil)
	r.AddCookie(cookieFor(t, ssoState{State: "the-state", Nonce: "the-nonce", Verifier: "the-verifier"}))
	h.SSOCallback(httptest.NewRecorder(), r)

	if p.gotCode != "the-code" || p.gotVerifier != "the-verifier" || p.gotNonce != "the-nonce" {
		t.Errorf("exchange got code=%q verifier=%q nonce=%q", p.gotCode, p.gotVerifier, p.gotNonce)
	}
}

// Routes are absent rather than present-and-refusing when SSO is off.
func TestSSORoutesAreNotMountedWhenUnconfigured(t *testing.T) {
	h := &Handlers{logger: discardLogger()}
	mux := chiStub{}
	h.ssoRoutes(&mux)
	if len(mux.paths) != 0 {
		t.Errorf("mounted %v with no provider configured", mux.paths)
	}
}

func TestAuthOptionsReportsWhetherSSOIsAvailable(t *testing.T) {
	off := &Handlers{logger: discardLogger()}
	rec := httptest.NewRecorder()
	off.AuthOptions(rec, httptest.NewRequest(http.MethodGet, "/api/auth/options", nil))
	if strings.Contains(rec.Body.String(), `"ssoEnabled":true`) {
		t.Error("reported SSO as available with no provider")
	}

	on := testHandlers(&stubProvider{})
	rec = httptest.NewRecorder()
	on.AuthOptions(rec, httptest.NewRequest(http.MethodGet, "/api/auth/options", nil))
	if !strings.Contains(rec.Body.String(), `"ssoEnabled":true`) {
		t.Error("did not report SSO as available with a provider configured")
	}
}
