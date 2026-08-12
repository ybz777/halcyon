package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseEmail_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user@example.com"},
		{"  User@Example.COM  ", "user@example.com"},
		{"name+tag@sub.example.co.uk", "name+tag@sub.example.co.uk"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			email, err := ParseEmail(tt.input)
			if err != nil {
				t.Fatalf("ParseEmail() error = %v", err)
			}
			if email.String() != tt.expected {
				t.Errorf("String() = %q, want %q", email.String(), tt.expected)
			}
		})
	}
}

func TestParseEmail_Invalid(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"not-an-email",
		"@example.com",
		"user@",
		"user@nodot",
		strings.Repeat("a", 250) + "@example.com",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			_, err := ParseEmail(input)
			if !errors.Is(err, ErrInvalidEmail) {
				t.Fatalf("error = %v, want ErrInvalidEmail", err)
			}
		})
	}
}

func TestEmail_Domain(t *testing.T) {
	t.Parallel()

	email, err := ParseEmail("user@halcyon.io")
	if err != nil {
		t.Fatalf("ParseEmail() error = %v", err)
	}

	if email.Domain() != "halcyon.io" {
		t.Errorf("Domain() = %q, want halcyon.io", email.Domain())
	}
}

func TestEmail_Equal(t *testing.T) {
	t.Parallel()

	a, _ := ParseEmail("a@example.com")
	b, _ := ParseEmail("A@EXAMPLE.COM")
	if !a.Equal(b) {
		t.Fatal("expected equal emails after normalization")
	}
}
