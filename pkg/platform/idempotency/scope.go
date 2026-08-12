package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Scope identifies the authenticated principal and tenant for an idempotency key.
type Scope struct {
	Service        string
	PrincipalType  string
	PrincipalID    string
	OrganizationID string
}

// ScopeKey returns a stable internal scope identifier.
func (s Scope) ScopeKey() string {
	parts := []string{
		sanitizePart(s.Service),
		sanitizePart(s.PrincipalType),
		hashPart(s.PrincipalID),
	}
	if org := strings.TrimSpace(s.OrganizationID); org != "" {
		parts = append(parts, hashPart(org))
	}
	return strings.Join(parts, ":")
}

// StorageKey builds the Redis key for a scoped idempotency key.
func StorageKey(prefix string, scope Scope, idempotencyKey string) string {
	return fmt.Sprintf("%s:idempotency:%s:%s",
		sanitizePart(prefix),
		scope.ScopeKey(),
		hashPart(idempotencyKey),
	)
}

func hashPart(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:16])
}

func sanitizePart(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ":", "_")
	if len(v) > 64 {
		v = v[:64]
	}
	if v == "" {
		return "unknown"
	}
	return v
}
