package migrations

import "embed"

// Files contains SQL migration files for platform messaging tables.
//
//go:embed *.sql
var Files embed.FS
