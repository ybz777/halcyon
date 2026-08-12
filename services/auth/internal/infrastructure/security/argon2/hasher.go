package argon2

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"golang.org/x/crypto/argon2"
)

// Params configures Argon2id hashing.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams returns production-oriented Argon2id defaults.
func DefaultParams() Params {
	return Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Hasher implements ports.PasswordHasher using Argon2id.
type Hasher struct {
	params Params
}

// NewHasher creates an Argon2id password hasher.
func NewHasher(params Params) *Hasher {
	return &Hasher{params: params}
}

// Hash hashes a plaintext password.
func (h *Hasher) Hash(_ context.Context, password string) (domain.PasswordHash, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return domain.PasswordHash{}, fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)

	encoded := encodeHash(h.params, salt, hash)
	return domain.ParsePasswordHash(encoded)
}

// Verify checks a password against a stored hash using constant-time comparison.
func (h *Hasher) Verify(_ context.Context, password string, stored domain.PasswordHash) (bool, error) {
	params, salt, expected, err := decodeHash(stored.String())
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(expected)),
	)

	if len(actual) != len(expected) {
		return false, nil
	}

	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

var _ ports.PasswordHasher = (*Hasher)(nil)

func encodeHash(params Params, salt, hash []byte) string {
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)
}

func decodeHash(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return Params{}, nil, nil, errors.New("invalid phc format")
	}
	if parts[1] != "argon2id" {
		return Params{}, nil, nil, errors.New("unsupported algorithm")
	}

	var params Params
	for _, segment := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(segment, "=", 2)
		if len(kv) != 2 {
			return Params{}, nil, nil, errors.New("invalid params")
		}
		value, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return Params{}, nil, nil, err
		}
		switch kv[0] {
		case "m":
			params.Memory = uint32(value)
		case "t":
			params.Iterations = uint32(value)
		case "p":
			params.Parallelism = uint8(value)
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, err
	}

	return params, salt, hash, nil
}
