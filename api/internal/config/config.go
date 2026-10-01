// Package config loads runtime configuration from the environment.
package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	TZ          string
	MailMode    string
	AdminEmail  string
}

func Load() Config {
	return Config{
		DatabaseURL: getenv("DATABASE_URL", "postgres://campaign_tracker_pro:campaign_tracker_pro_dev_pw@localhost:5432/campaign_tracker_pro?sslmode=disable"),
		Port:        getenv("PORT", "8090"),
		TZ:          getenv("TZ", "Asia/Kolkata"),
		MailMode:    getenv("MAIL_MODE", "log"),
		AdminEmail:  getenv("ADMIN_EMAIL", "anishatiwari695@gmail.com"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
