package ids

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// Generate creates a prefixed, URL-safe identifier (e.g. usr_abc123).
func Generate(prefix string) (string, error) {
	if prefix == "" {
		return "", fmt.Errorf("prefix is required")
	}
	if !strings.HasSuffix(prefix, "_") {
		prefix += "_"
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(b)
	return prefix + encoded, nil
}

// MustGenerate calls Generate and panics on error. Use only in tests.
func MustGenerate(prefix string) string {
	id, err := Generate(prefix)
	if err != nil {
		panic(err)
	}
	return id
}
