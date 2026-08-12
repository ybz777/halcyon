package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SecretPrefix       = "whsec_"
	DefaultMaxSkew     = 5 * time.Minute
	SignatureHeader    = "X-Webhook-Signature"
	TimestampHeader    = "X-Webhook-Timestamp"
	EventIDHeader      = "X-Webhook-Event-ID"
	EventTypeHeader    = "X-Webhook-Event-Type"
	SignatureVersion   = "v1"
)

// GenerateSecret creates a webhook signing secret shown once to customers.
func GenerateSecret() (plaintext string, prefix string, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate secret: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	plaintext = SecretPrefix + encoded
	prefix = plaintext
	if len(plaintext) > 12 {
		prefix = plaintext[:12]
	}
	hash = HashSecret(plaintext)
	return plaintext, prefix, hash, nil
}

// HashSecret returns a hex-encoded SHA-256 hash for storage.
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// VerifySecret compares a presented secret with stored hash.
func VerifySecret(presented, storedHash string) bool {
	return hmac.Equal([]byte(HashSecret(presented)), []byte(storedHash))
}

// SignPayload produces an HMAC-SHA256 signature over timestamp.raw_payload.
func SignPayload(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return SignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature validates webhook signature and timestamp skew.
func VerifySignature(secret string, timestamp int64, payload []byte, signature string, now time.Time, maxSkew time.Duration) error {
	if maxSkew <= 0 {
		maxSkew = DefaultMaxSkew
	}
	ts := time.Unix(timestamp, 0).UTC()
	if now.Sub(ts) > maxSkew || ts.Sub(now) > maxSkew {
		return fmt.Errorf("timestamp outside tolerance")
	}
	expected := SignPayload(secret, timestamp, payload)
	if !hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature))) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
