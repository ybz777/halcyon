package argon2_test

import (
	"context"
	"testing"

	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/argon2"
)

func TestHasher_HashAndVerify(t *testing.T) {
	t.Parallel()

	hasher := argon2.NewHasher(argon2.Params{
		Memory:      64 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})

	hash, err := hasher.Hash(context.Background(), "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, err := hasher.Verify(context.Background(), "correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("expected password verification to succeed")
	}

	ok, err = hasher.Verify(context.Background(), "wrong-password", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if ok {
		t.Fatal("expected password verification to fail")
	}
}
