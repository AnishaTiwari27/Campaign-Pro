package database

import "testing"

func TestMigrateURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			// The case that matters: a pooled Neon URL must not carry the
			// session advisory lock through PgBouncer.
			"neon pooled is routed to the direct host",
			"postgresql://u:p@ep-cool-sun-a1b2c3-pooler.ap-southeast-1.aws.neon.tech/neondb?sslmode=require",
			"pgx5://u:p@ep-cool-sun-a1b2c3.ap-southeast-1.aws.neon.tech/neondb?sslmode=require",
		},
		{
			"neon direct is left alone",
			"postgresql://u:p@ep-cool-sun-a1b2c3.ap-southeast-1.aws.neon.tech/neondb?sslmode=require",
			"pgx5://u:p@ep-cool-sun-a1b2c3.ap-southeast-1.aws.neon.tech/neondb?sslmode=require",
		},
		{
			"postgres:// scheme is accepted too",
			"postgres://u:p@localhost:5432/db?sslmode=disable",
			"pgx5://u:p@localhost:5432/db?sslmode=disable",
		},
		{
			// Only Neon hosts are rewritten; another provider's "-pooler."
			// is its own business.
			"a non-neon pooler host is untouched",
			"postgres://u:p@db-pooler.example.com:6432/db",
			"pgx5://u:p@db-pooler.example.com:6432/db",
		},
		{
			"query string and credentials survive intact",
			"postgres://user:p%40ss@host/db?sslmode=require&application_name=ctp",
			"pgx5://user:p%40ss@host/db?sslmode=require&application_name=ctp",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := migrateURL(c.in); got != c.want {
				t.Fatalf("migrateURL(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}
