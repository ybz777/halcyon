package servicetoken

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	HeaderName = "X-Halcyon-Service-Token"
)

var (
	ErrInvalidKey       = errors.New("invalid service token key")
	ErrInvalidToken     = errors.New("invalid service token")
	ErrExpiredToken     = errors.New("service token expired")
	ErrInvalidAlgorithm = errors.New("unexpected signing algorithm")
)

// Claims are short-lived internal service credentials.
type Claims struct {
	ServiceID string `json:"sid"`
	jwt.RegisteredClaims
}

// Signer issues service-to-service JWTs.
type Signer struct {
	privateKey *rsa.PrivateKey
	keyID      string
	issuer     string
	audience   string
	ttl        time.Duration
}

// NewSigner creates a signer from a PEM-encoded RSA private key.
func NewSigner(privateKeyPEM, keyID, issuer, audience string, ttl time.Duration) (*Signer, error) {
	key, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("%w: key id required", ErrInvalidKey)
	}
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, fmt.Errorf("%w: issuer and audience required", ErrInvalidKey)
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Signer{
		privateKey: key,
		keyID:      keyID,
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
	}, nil
}

// Sign creates a signed service token.
func (s *Signer) Sign(serviceID string) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		ServiceID: serviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   serviceID,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	})
	token.Header["kid"] = s.keyID
	return token.SignedString(s.privateKey)
}

// Validator verifies service tokens.
type Validator struct {
	publicKey *rsa.PublicKey
	issuer    string
	audience  string
}

// NewValidator creates a validator from a PEM-encoded RSA public key.
func NewValidator(publicKeyPEM, issuer, audience string) (*Validator, error) {
	key, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}
	return &Validator{publicKey: key, issuer: issuer, audience: audience}, nil
}

// Validate parses and validates a service token.
func (v *Validator) Validate(tokenString string) (Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodRS256 {
			return nil, ErrInvalidAlgorithm
		}
		return v.publicKey, nil
	}, jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrExpiredToken
		}
		return Claims{}, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}
	return *claims, nil
}

// PublicKeyPEM returns the PKIX public key for JWKS publication.
func PublicKeyPEM(privateKeyPEM string) (string, error) {
	key, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("%w: invalid PEM", ErrInvalidKey)
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%w: not RSA", ErrInvalidKey)
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("%w: unsupported key type %q", ErrInvalidKey, block.Type)
	}
}

func parseRSAPublicKey(pemData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("%w: invalid PEM", ErrInvalidKey)
	}
	switch block.Type {
	case "PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("%w: not RSA", ErrInvalidKey)
		}
		return rsaKey, nil
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, fmt.Errorf("%w: unsupported key type %q", ErrInvalidKey, block.Type)
	}
}
