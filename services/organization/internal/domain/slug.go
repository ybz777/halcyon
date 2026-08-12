package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slug is a globally unique organization slug.
type Slug struct {
	value string
}

// SlugFromName derives a slug candidate from an organization name.
func SlugFromName(name OrganizationName) (Slug, error) {
	raw := strings.ToLower(name.String())
	raw = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-")
	if raw == "" {
		return Slug{}, fmt.Errorf("%w: cannot derive slug", ErrInvalidSlug)
	}
	return ParseSlug(raw)
}

// ParseSlug validates a slug value.
func ParseSlug(raw string) (Slug, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return Slug{}, fmt.Errorf("%w: empty", ErrInvalidSlug)
	}
	if !slugPattern.MatchString(raw) {
		return Slug{}, fmt.Errorf("%w: invalid format", ErrInvalidSlug)
	}
	return Slug{value: raw}, nil
}

func (s Slug) String() string { return s.value }
func (s Slug) IsZero() bool   { return s.value == "" }
