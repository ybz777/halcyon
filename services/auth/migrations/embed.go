package migrations

import "embed"

// Files contains SQL migration files for the auth service database.
//
//go:embed *.sql
var Files embed.FS
