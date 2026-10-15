// Package config loads runtime configuration from the environment.
package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// devDatabaseURL matches what `make db-create` sets up, so a checkout runs
// with no configuration at all. It is only ever right on a developer's own
// machine.
const devDatabaseURL = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"

// ErrDatabaseURLMissing is returned when the local development fallback is in
// use somewhere it cannot possibly work. The message names the fix rather than
// the symptom: the symptom is a connection refused to 127.0.0.1 as a user that
// exists only on a developer's laptop, which reads as a broken database.
var ErrDatabaseURLMissing = errors.New(
	"DATABASE_URL is not set, and this process is running on a managed host " +
		"where the local development fallback cannot work. Set DATABASE_URL on " +
		"the service to your Postgres URL, including ?sslmode=require. On Render " +
		"that is the service's Environment tab: render.yaml marks the key " +
		"sync: false, so a Blueprint sync deliberately never supplies the value")

// managedHostMarkers are environment variables that container platforms set on
// every instance they run. Their presence means the process is not on a
// developer's machine, which turns the database fallback from "probably wrong"
// into "certainly wrong".
//
// A heuristic, and deliberately a short one: it is a safety net that converts a
// confusing failure into a clear one, not a boundary anything relies on. A host
// that is not listed simply gets the old behaviour — the warning, then the
// connection error — so adding one is a one-line improvement and never a fix
// for something that would otherwise be broken.
var managedHostMarkers = []string{
	"RENDER",                     // Render
	"FLY_APP_NAME",               // Fly.io
	"DYNO",                       // Heroku
	"K_SERVICE",                  // Google Cloud Run
	"WEBSITE_INSTANCE_ID",        // Azure App Service
	"ECS_CONTAINER_METADATA_URI", // AWS ECS/Fargate
	"ECS_CONTAINER_METADATA_URI_V4",
}

type Config struct {
	DatabaseURL string
	Port        string
	TZ          string
	MailMode    string
	AdminEmail  string
	// SecureCookies must be true anywhere served over HTTPS; local dev
	// is http, so it is off by default and switched on in production.
	SecureCookies bool
	// DatabaseURLDefaulted reports that DATABASE_URL was unset and the local
	// development fallback is in use. Worth saying out loud: in a deployment
	// that fallback makes the process dial localhost as a user that does not
	// exist there, and "connection refused to 127.0.0.1" is a confusing way
	// to learn that a variable is missing.
	DatabaseURLDefaulted bool
	// ManagedHost reports that a container platform's marker is present, so
	// the defaults meant for a laptop are known to be wrong here.
	ManagedHost bool
	// AllowSignup decides whether the public signup page and endpoint
	// exist. On by default, so a fresh deployment is usable without
	// reaching for the database; set ALLOW_SIGNUP=false to close it once
	// the accounts that should exist do.
	AllowSignup bool
	// RateAPIPerMinute and RateAuthPerMinute are per-IP request budgets.
	//
	// Tunable without a code change because the right number depends on
	// where the traffic comes from, which the code cannot know: a single
	// user is nowhere near either limit, but an office behind one NAT
	// address shares a bucket, and so does a CI suite. If real users
	// start seeing 429s, this is the knob — not a redeploy of the
	// limiter.
	//
	// The auth default started at 30/min and was too tight: the e2e
	// suite makes about thirty-four sign-in attempts a minute from one
	// address and began failing on 429s, which is a fair proxy for a
	// team behind one NAT address. 120/min is still two per second,
	// which is nowhere near fast enough to be useful for guessing, and
	// a per-IP limit was never the real defence against a targeted
	// attacker anyway — they rotate addresses. It stops one noisy
	// source, which is what it is for.
	RateAPIPerMinute  int
	RateAuthPerMinute int
	// SMTP settings for scheduled report digests. MailMode picks the
	// implementation: "log" writes the message to the log and is the
	// default, "smtp" sends it. Deliberately opt-in — a misconfigured
	// host that silently mails nobody is worse than one that plainly
	// says it is in log mode.
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	MailFrom string
	// Google Ads ingestion. All five are required together: the
	// developer token is issued per manager account on application, and
	// the OAuth client and refresh token belong to a Google Cloud
	// project, so there is no useful partial configuration.
	GoogleAdsDeveloperToken string
	GoogleAdsClientID       string
	GoogleAdsClientSecret   string
	GoogleAdsRefreshToken   string
	GoogleAdsCustomerID     string
	// Only needed when the customer sits under a manager account.
	GoogleAdsLoginCustomerID string

	// Meta Marketing API. The access token is the credential itself —
	// there is no refresh exchange — so it expires on Meta's schedule
	// and has to be reissued rather than renewed.
	MetaAccessToken string
	MetaAdAccountID string

	// Single sign-on, OIDC. Off unless an issuer is set: a deployment
	// with no identity provider should keep the password form rather
	// than show an SSO button that leads nowhere.
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	// OIDCRedirectURL must match what is registered with the provider
	// exactly, including scheme and trailing path, or the provider
	// refuses the exchange.
	OIDCRedirectURL string
	// OIDCOrgID is the organisation an SSO user joins.
	//
	// One issuer to one organisation, which is enough for a deployment
	// with a single identity provider and is where this stops. Real
	// multi-tenant SSO needs an issuer-to-org registry so each agency
	// can bring its own provider; that is a table and a admin screen,
	// not a config line, and is deliberately not half-built here.
	OIDCOrgID string

	// SignupOrgID is the organisation a self-registered account joins.
	//
	// Signup is a public endpoint, so there is no session to read a
	// tenant from and one has to be named here. It answers for the
	// password form exactly what OIDCOrgID answers for SSO, and for the
	// same reason: an account has to belong to an organisation, and
	// users.org_id is a foreign key, so "none" is not an option the
	// database will accept.
	SignupOrgID string

	// The model that writes signal explanations. Optional: with no key
	// the explanations are assembled from the figures instead, which is
	// plainer and still true.
	LLMAPIKey string
	LLMModel  string
}

func Load() Config {
	databaseURL := cleanDSN(os.Getenv("DATABASE_URL"))
	return Config{
		DatabaseURL:          or(databaseURL, devDatabaseURL),
		DatabaseURLDefaulted: databaseURL == "",
		ManagedHost:          managedHost(),
		Port:                 getenv("PORT", "8090"),
		TZ:                   getenv("TZ", "Asia/Kolkata"),
		MailMode:             getenv("MAIL_MODE", "log"),
		AdminEmail:           getenv("ADMIN_EMAIL", "anishatiwari695@gmail.com"),
		SecureCookies:        getenv("SECURE_COOKIES", "false") == "true",
		AllowSignup:          getenv("ALLOW_SIGNUP", "true") == "true",
		RateAPIPerMinute:     getint("RATE_API_PER_MINUTE", 600),
		RateAuthPerMinute:    getint("RATE_AUTH_PER_MINUTE", 120),
		SMTPHost:             os.Getenv("SMTP_HOST"),
		SMTPPort:             getenv("SMTP_PORT", "587"),
		SMTPUser:             os.Getenv("SMTP_USERNAME"),
		SMTPPass:             os.Getenv("SMTP_PASSWORD"),
		MailFrom:             getenv("MAIL_FROM", "reports@campaigntracker.local"),

		GoogleAdsDeveloperToken:  os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN"),
		GoogleAdsClientID:        os.Getenv("GOOGLE_ADS_CLIENT_ID"),
		GoogleAdsClientSecret:    os.Getenv("GOOGLE_ADS_CLIENT_SECRET"),
		GoogleAdsRefreshToken:    os.Getenv("GOOGLE_ADS_REFRESH_TOKEN"),
		GoogleAdsCustomerID:      os.Getenv("GOOGLE_ADS_CUSTOMER_ID"),
		GoogleAdsLoginCustomerID: os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID"),

		MetaAccessToken: os.Getenv("META_ACCESS_TOKEN"),
		MetaAdAccountID: os.Getenv("META_AD_ACCOUNT_ID"),

		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		OIDCOrgID:        getenv("OIDC_ORG_ID", "default"),
		SignupOrgID:      getenv("SIGNUP_ORG_ID", "default"),

		LLMAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		LLMModel:  getenv("LLM_MODEL", "claude-sonnet-5"),
	}
}

// SSOEnabled reports whether single sign-on is configured. All four
// values are needed: a partial configuration is a misconfiguration, and
// offering the button anyway would send people to a provider that
// rejects the request.
func (c Config) SSOEnabled() bool {
	return c.OIDCIssuer != "" && c.OIDCClientID != "" &&
		c.OIDCClientSecret != "" && c.OIDCRedirectURL != ""
}

// SSOProviderName is the provider's name, for the label on the sign-in
// button. Derived from the issuer rather than configured separately, so
// the two cannot disagree about who the button sends you to.
//
// A known host gets its name; anything else gets the generic wording.
// Guessing from an unrecognised domain would be worse than generic: a
// button that names the wrong company is a reason not to click it.
func (c Config) SSOProviderName() string {
	host := c.OIDCIssuer
	if u, err := url.Parse(c.OIDCIssuer); err == nil && u.Host != "" {
		host = u.Host
	}
	host = strings.ToLower(host)
	switch {
	case host == "accounts.google.com":
		return "Google"
	case under(host, "microsoftonline.com"), under(host, "windows.net"):
		return "Microsoft"
	case under(host, "okta.com"), under(host, "oktapreview.com"):
		return "Okta"
	case under(host, "auth0.com"):
		return "Auth0"
	case under(host, "onelogin.com"):
		return "OneLogin"
	}
	return "single sign-on"
}

// under reports whether host is domain itself or a subdomain of it.
// A bare suffix test would also match "evilmicrosoftonline.com", which
// is the one answer this function must never give.
func under(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// Validate reports configuration that cannot work, as distinct from
// configuration that is merely unusual. It is checked before the first
// connection is attempted, so a missing variable is reported as itself
// instead of as whatever the resulting dial happens to fail with.
func (c Config) Validate() error {
	if c.DatabaseURLDefaulted && c.ManagedHost {
		return ErrDatabaseURLMissing
	}
	return nil
}

// cleanDSN removes what a copy-paste into a dashboard field tends to carry:
// surrounding whitespace, and a matching pair of quotes taken along from a
// shell command like DATABASE_URL='postgres://...'. A platform stores the
// literal characters it was given, and pgx then fails on a DSN it cannot
// parse — an error that never mentions quoting. A valid DSN never begins and
// ends with a quote, so removing one pair cannot discard a working value.
func cleanDSN(v string) string {
	v = strings.TrimSpace(v)
	for _, quote := range []string{`"`, "'"} {
		if len(v) >= 2 && strings.HasPrefix(v, quote) && strings.HasSuffix(v, quote) {
			return strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}

func managedHost() bool {
	for _, key := range managedHostMarkers {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

func getenv(key, def string) string {
	return or(os.Getenv(key), def)
}

// getint reads a positive integer, falling back to def for anything that
// is not one. A malformed limit must not become zero: a zero budget
// would refuse every request, which is a worse failure than ignoring a
// typo in an environment variable.
func getint(key string, def int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func or(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
