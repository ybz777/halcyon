package domain

import (
	"errors"
	"testing"
)

const validBcryptHash = "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.G2oYbOuSO.8K6O"

func TestParsePasswordHash_ValidBcrypt(t *testing.T) {
	t.Parallel()

	hash, err := ParsePasswordHash(validBcryptHash)
	if err != nil {
		t.Fatalf("ParsePasswordHash() error = %v", err)
	}

	if hash.String() != validBcryptHash {
		t.Errorf("String() mismatch")
	}
}

func TestParsePasswordHash_ValidArgon2(t *testing.T) {
	t.Parallel()

	raw := "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$hashvaluehashvaluehashvaluehash"
	hash, err := ParsePasswordHash(raw)
	if err != nil {
		t.Fatalf("ParsePasswordHash() error = %v", err)
	}

	if hash.IsZero() {
		t.Fatal("expected non-zero hash")
	}
}

func TestParsePasswordHash_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"plaintext", "not-a-hash"},
		{"too short bcrypt", "$2b$12$short"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParsePasswordHash(tt.input)
			if !errors.Is(err, ErrInvalidPasswordHash) {
				t.Fatalf("error = %v, want ErrInvalidPasswordHash", err)
			}
		})
	}
}

func TestPasswordHash_Equal(t *testing.T) {
	t.Parallel()

	a, _ := ParsePasswordHash(validBcryptHash)
	b, _ := ParsePasswordHash(validBcryptHash)
	if !a.Equal(b) {
		t.Fatal("expected equal hashes")
	}
}

func TestPasswordHash_Bytes(t *testing.T) {
	t.Parallel()

	hash, _ := ParsePasswordHash(validBcryptHash)
	if string(hash.Bytes()) != validBcryptHash {
		t.Fatal("Bytes() mismatch")
	}
}
