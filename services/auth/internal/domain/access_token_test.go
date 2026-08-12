package domain

import (
	"errors"
	"testing"
)

func TestParseAccessToken(t *testing.T) {
	t.Parallel()

	token, err := ParseAccessToken("eyJhbGciOiJIUzI1NiJ9.payload.signature")
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if token.IsZero() {
		t.Fatal("expected non-zero token")
	}
}

func TestParseAccessToken_Empty(t *testing.T) {
	t.Parallel()

	_, err := ParseAccessToken("")
	if !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("error = %v, want ErrInvalidAccessToken", err)
	}
}

func TestNewRefreshToken(t *testing.T) {
	t.Parallel()

	token, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	if token.IsZero() {
		t.Fatal("expected non-zero refresh token")
	}
}

func TestParseRefreshToken_TooShort(t *testing.T) {
	t.Parallel()

	_, err := ParseRefreshToken("short")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("error = %v, want ErrInvalidRefreshToken", err)
	}
}
