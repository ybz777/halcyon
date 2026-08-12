package jwks

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// KeySet is a JSON Web Key Set.
type KeySet struct {
	Keys []Key `json:"keys"`
}

// Key is a JSON Web Key for HS256 verification.
type Key struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	K   string `json:"k"`
}

// Provider builds JWKS from symmetric signing material.
type Provider struct {
	keyID  string
	secret []byte
}

// NewProvider creates a JWKS provider for HS256 tokens.
func NewProvider(keyID string, secret []byte) *Provider {
	return &Provider{keyID: keyID, secret: secret}
}

// KeySet returns the current verification keys.
func (p *Provider) KeySet() KeySet {
	sum := sha256.Sum256(p.secret)
	kid := p.keyID
	if kid == "" {
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	return KeySet{Keys: []Key{{
		Kty: "oct",
		Kid: kid,
		Alg: "HS256",
		Use: "sig",
		K:   base64.RawURLEncoding.EncodeToString(p.secret),
	}}}
}

// Marshal returns JSON-encoded JWKS.
func (p *Provider) Marshal() ([]byte, error) {
	return json.Marshal(p.KeySet())
}
