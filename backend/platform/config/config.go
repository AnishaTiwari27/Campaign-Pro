// Package config loads runtime configuration from the environment.
package config

import "os"

// devDatabaseURL matches what `make db-create` sets up, so a checkout runs
// with no configuration at all. It is only ever right on a developer's own
// machine.
const devDatabaseURL = "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"

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
}

func Load() Config {
	return Config{
		DatabaseURL:          getenv("DATABASE_URL", devDatabaseURL),
		DatabaseURLDefaulted: os.Getenv("DATABASE_URL") == "",
		Port:                 getenv("PORT", "8090"),
		TZ:                   getenv("TZ", "Asia/Kolkata"),
		MailMode:             getenv("MAIL_MODE", "log"),
		AdminEmail:           getenv("ADMIN_EMAIL", "anishatiwari695@gmail.com"),
		SecureCookies:        getenv("SECURE_COOKIES", "false") == "true",
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
