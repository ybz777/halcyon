package domain

import (
	"errors"
	"testing"
)

func TestParseUserID_Valid(t *testing.T) {
	t.Parallel()

	id, err := ParseUserID("usr_abcdefghijklmnopqrst")
	if err != nil {
		t.Fatalf("ParseUserID() error = %v", err)
	}

	if id.String() != "usr_abcdefghijklmnopqrst" {
		t.Errorf("String() = %q", id.String())
	}
}

func TestParseUserID_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"wrong prefix", "org_abcdefghijklmnopqrst"},
		{"too short", "usr_short"},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseUserID(tt.input)
			if !errors.Is(err, ErrInvalidUserID) {
				t.Fatalf("error = %v, want ErrInvalidUserID", err)
			}
		})
	}
}

func TestNewUserID_Unique(t *testing.T) {
	t.Parallel()

	id1, err := NewUserID()
	if err != nil {
		t.Fatalf("NewUserID() error = %v", err)
	}

	id2, err := NewUserID()
	if err != nil {
		t.Fatalf("NewUserID() error = %v", err)
	}

	if id1.Equal(id2) {
		t.Fatal("expected unique user ids")
	}

	if !ParseUserIDValid(id1.String()) {
		t.Errorf("generated id %q failed validation", id1.String())
	}
}

func ParseUserIDValid(raw string) bool {
	_, err := ParseUserID(raw)
	return err == nil
}

func TestUserID_IsZero(t *testing.T) {
	t.Parallel()

	var zero UserID
	if !zero.IsZero() {
		t.Fatal("zero value should be zero")
	}

	id, _ := NewUserID()
	if id.IsZero() {
		t.Fatal("generated id should not be zero")
	}
}
