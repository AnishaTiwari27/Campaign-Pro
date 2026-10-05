// Package db holds the one shared schema: the golang-migrate sources, the
// sqlc query sources, and sqlc's generated code. This file embeds the
// migrations so the server can apply them itself, which is what lets a
// deploy to a blank database come up with a schema and no separate tooling
// in the image.
//
// The same files still drive `make migrate`, and both paths use
// golang-migrate, so they share its schema_migrations bookkeeping and can
// never disagree about which version is applied.
package db

import "embed"

//go:embed migrations/*.sql
var migrations embed.FS

// Migrations is the embedded migration set, for a golang-migrate iofs source.
var Migrations = migrations
