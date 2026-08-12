package domain

import (
	"fmt"
	"strings"
)

// AccessToken is a JWT access token value object.
type AccessToken struct {
	value string
}

// ParseAccessToken validates a bearer access token string.
func ParseAccessToken(raw string) (AccessToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AccessToken{}, fmt.Errorf("%w: empty", ErrInvalidAccessToken)
	}
	return AccessToken{value: raw}, nil
}

// String returns the token value.
func (t AccessToken) String() string {
	return t.value
}

// IsZero reports whether the token is unset.
func (t AccessToken) IsZero() bool {
	return t.value == ""
}

// Equal reports whether two access tokens are identical.
func (t AccessToken) Equal(other AccessToken) bool {
	return t.value == other.value
}
