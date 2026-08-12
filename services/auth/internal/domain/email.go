package domain

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const maxEmailLength = 254

// Email is an immutable, normalized email address value object.
type Email struct {
	value string
}

// ParseEmail validates and normalizes an email address.
func ParseEmail(raw string) (Email, error) {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	if normalized == "" {
		return Email{}, fmt.Errorf("%w: empty", ErrInvalidEmail)
	}

	if utf8.RuneCountInString(normalized) > maxEmailLength {
		return Email{}, fmt.Errorf("%w: exceeds %d characters", ErrInvalidEmail, maxEmailLength)
	}

	addr, err := mail.ParseAddress(normalized)
	if err != nil {
		return Email{}, fmt.Errorf("%w: %v", ErrInvalidEmail, err)
	}

	parsed := strings.ToLower(strings.TrimSpace(addr.Address))
	if parsed != normalized {
		return Email{}, fmt.Errorf("%w: address mismatch after parsing", ErrInvalidEmail)
	}

	if !strings.Contains(parsed, "@") || strings.HasPrefix(parsed, "@") || strings.HasSuffix(parsed, "@") {
		return Email{}, fmt.Errorf("%w: malformed address", ErrInvalidEmail)
	}

	local, domain, ok := strings.Cut(parsed, "@")
	if !ok || local == "" || domain == "" || !strings.Contains(domain, ".") {
		return Email{}, fmt.Errorf("%w: malformed address", ErrInvalidEmail)
	}

	return Email{value: parsed}, nil
}

// String returns the normalized email address.
func (e Email) String() string {
	return e.value
}

// IsZero reports whether the Email is unset.
func (e Email) IsZero() bool {
	return e.value == ""
}

// Equal reports whether two Email values are the same.
func (e Email) Equal(other Email) bool {
	return e.value == other.value
}

// Domain returns the domain portion of the email address.
func (e Email) Domain() string {
	_, domain, _ := strings.Cut(e.value, "@")
	return domain
}
