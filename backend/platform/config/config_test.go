package config

import (
	"errors"
	"testing"
)

// clearHostMarkers makes a test independent of whatever platform variables
// happen to be set where it runs, including in CI.
func clearHostMarkers(t *testing.T) {
	t.Helper()
	for _, key := range managedHostMarkers {
		t.Setenv(key, "")
	}
}

func TestCleanDSN(t *testing.T) {
	const dsn = "postgres://u:p@ep-cool-sun-a1b2c3.ap-southeast-1.aws.neon.tech/neondb?sslmode=require"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"a bare URL is untouched", dsn, dsn},
		{
			// The paste that matters: copied out of the README's createuser
			// command, quotes and all, into a dashboard field that stores
			// exactly what it is given.
			"single quotes from a shell command are removed",
			"'" + dsn + "'",
			dsn,
		},
		{"double quotes are removed", `"` + dsn + `"`, dsn},
		{"surrounding whitespace and a trailing newline are removed", "  " + dsn + "\n", dsn},
		{"quotes inside whitespace are still removed", " '" + dsn + "' ", dsn},
		{"an unmatched leading quote is left alone", `"` + dsn, `"` + dsn},
		{"an empty value stays empty", "   ", ""},
		{"a lone quote is not treated as a pair", `"`, `"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := cleanDSN(c.in); got != c.want {
				t.Fatalf("cleanDSN(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	const dsn = "postgres://u:p@db.example.com/app?sslmode=require"

	t.Run("an explicit URL is used as given", func(t *testing.T) {
		clearHostMarkers(t)
		t.Setenv("DATABASE_URL", dsn)
		cfg := Load()
		if cfg.DatabaseURL != dsn {
			t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dsn)
		}
		if cfg.DatabaseURLDefaulted {
			t.Fatal("DatabaseURLDefaulted = true, want false")
		}
	})

	t.Run("a quoted URL is cleaned, not treated as absent", func(t *testing.T) {
		clearHostMarkers(t)
		t.Setenv("DATABASE_URL", "'"+dsn+"'")
		cfg := Load()
		if cfg.DatabaseURL != dsn {
			t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dsn)
		}
		if cfg.DatabaseURLDefaulted {
			t.Fatal("DatabaseURLDefaulted = true, want false")
		}
	})

	t.Run("unset falls back to the development database", func(t *testing.T) {
		clearHostMarkers(t)
		t.Setenv("DATABASE_URL", "")
		cfg := Load()
		if cfg.DatabaseURL != devDatabaseURL {
			t.Fatalf("DatabaseURL = %q, want the dev fallback", cfg.DatabaseURL)
		}
		if !cfg.DatabaseURLDefaulted {
			t.Fatal("DatabaseURLDefaulted = false, want true")
		}
	})

	t.Run("whitespace only counts as unset", func(t *testing.T) {
		clearHostMarkers(t)
		t.Setenv("DATABASE_URL", "   \n")
		cfg := Load()
		if !cfg.DatabaseURLDefaulted {
			t.Fatal("DatabaseURLDefaulted = false, want true")
		}
	})
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{
			// The deploy this guard exists for: no DATABASE_URL on Render, so
			// the fallback would dial a localhost Postgres that is not there.
			"the dev fallback on a managed host is fatal",
			Config{DatabaseURLDefaulted: true, ManagedHost: true},
			ErrDatabaseURLMissing,
		},
		{
			"the dev fallback on a laptop is fine",
			Config{DatabaseURLDefaulted: true, ManagedHost: false},
			nil,
		},
		{
			"an explicit URL on a managed host is fine",
			Config{DatabaseURLDefaulted: false, ManagedHost: true},
			nil,
		},
		{
			"an explicit URL on a laptop is fine",
			Config{DatabaseURLDefaulted: false, ManagedHost: false},
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.cfg.Validate(); !errors.Is(err, c.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestLoadManagedHost(t *testing.T) {
	t.Run("no marker means a developer machine", func(t *testing.T) {
		clearHostMarkers(t)
		if Load().ManagedHost {
			t.Fatal("ManagedHost = true with no platform marker set")
		}
	})

	// Every marker is checked, so a host added to the list without a test is
	// visible as a gap rather than silently untested.
	for _, key := range managedHostMarkers {
		t.Run(key+" marks a managed host", func(t *testing.T) {
			clearHostMarkers(t)
			t.Setenv(key, "true")
			if !Load().ManagedHost {
				t.Fatalf("ManagedHost = false with %s set", key)
			}
		})
	}
}
