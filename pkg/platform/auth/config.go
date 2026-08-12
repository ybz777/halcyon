package auth

import (
	"fmt"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

// JWTSettings holds validated JWT validation settings for downstream services.
type JWTSettings struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// ServiceAuthSettings holds gateway trust configuration for internal services.
type ServiceAuthSettings struct {
	PublicKeyPEM string
	Issuer       string
	Audience     string
}

// LoadJWTSettings reads JWT validation settings from centralized configuration.
func LoadJWTSettings(env *platformconfig.Env, issuerDefault string) (JWTSettings, error) {
	if env == nil {
		env = platformconfig.FromEnv()
	}
	secret := env.Secret("JWT_SECRET")
	issuer := env.String("JWT_ISSUER", issuerDefault)
	if issuer == "" {
		issuer = issuerDefault
	}
	return JWTSettings{
		Secret:   []byte(secret.Value()),
		Issuer:   issuer,
		Audience: env.String("JWT_AUDIENCE", ""),
	}, nil
}

// LoadServiceAuthSettings reads service-to-service trust settings.
func LoadServiceAuthSettings(env *platformconfig.Env) (ServiceAuthSettings, error) {
	if env == nil {
		env = platformconfig.FromEnv()
	}
	publicKey := env.Secret("SERVICE_TOKEN_PUBLIC_KEY").Value()
	if publicKey == "" {
		privateKey := env.Secret("SERVICE_PRIVATE_KEY").Value()
		if privateKey != "" {
			derived, err := servicetoken.PublicKeyPEM(privateKey)
			if err != nil {
				return ServiceAuthSettings{}, fmt.Errorf("derive service token public key: %w", err)
			}
			publicKey = derived
		}
	}
	return ServiceAuthSettings{
		PublicKeyPEM: publicKey,
		Issuer:       env.String("SERVICE_TOKEN_ISSUER", "gateway"),
		Audience:     env.String("SERVICE_TOKEN_AUDIENCE", "halcyon-internal"),
	}, nil
}

// NewHS256ValidatorFromEnv builds a validator using centralized configuration.
func NewHS256ValidatorFromEnv(issuerDefault string) (*HS256Validator, error) {
	settings, err := LoadJWTSettings(platformconfig.FromEnv(), issuerDefault)
	if err != nil {
		return nil, err
	}
	return NewHS256Validator(HS256Config{
		Secret:   settings.Secret,
		Issuer:   settings.Issuer,
		Audience: settings.Audience,
	})
}

// NewGatewayTrustFromEnv builds gateway trust validation from centralized configuration.
func NewGatewayTrustFromEnv(service string) (*GatewayTrust, error) {
	settings, err := LoadServiceAuthSettings(platformconfig.FromEnv())
	if err != nil {
		return nil, err
	}
	if settings.PublicKeyPEM == "" {
		return NewGatewayTrust(nil, NewPrometheusMetrics(service)), nil
	}
	validator, err := servicetoken.NewValidator(settings.PublicKeyPEM, settings.Issuer, settings.Audience)
	if err != nil {
		return nil, err
	}
	return NewGatewayTrust(validator, NewPrometheusMetrics(service)), nil
}
