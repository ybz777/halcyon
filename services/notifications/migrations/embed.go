package migrations

import "embed"

// Files contains SQL migrations for the notifications service.
//
//go:embed *.sql
var Files embed.FS
