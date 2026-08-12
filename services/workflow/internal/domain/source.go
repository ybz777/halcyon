package domain

import (
	"fmt"
	"strings"
)

// Source identifies where an incident originated.
type Source string

// ParseSource validates an incident source.
func ParseSource(raw string) (Source, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 128 {
		return "", fmt.Errorf("%w: %q", ErrInvalidSource, raw)
	}
	return Source(raw), nil
}

func (s Source) String() string { return string(s) }

// Title is the incident headline.
type Title string

// ParseTitle validates an incident title.
func ParseTitle(raw string) (Title, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 256 {
		return "", fmt.Errorf("%w", ErrInvalidTitle)
	}
	return Title(raw), nil
}

func (t Title) String() string { return string(t) }
func (t Title) IsZero() bool   { return t == "" }

// Description holds incident details.
type Description string

// ParseDescription validates incident description text.
func ParseDescription(raw string) (Description, error) {
	if len(raw) > 8192 {
		return "", fmt.Errorf("%w: too long", ErrInvalidDescription)
	}
	return Description(strings.TrimSpace(raw)), nil
}

func (d Description) String() string { return string(d) }
