package config

import (
	"fmt"

	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

// LoadServiceAuth reads service-to-service gateway trust configuration.
func LoadServiceAuth(env *Env) (ServiceAuth, error) {
	if env == nil {
		env = FromEnv()
	}
	publicKey := env.Secret("SERVICE_TOKEN_PUBLIC_KEY")
	if !publicKey.IsSet() {
		privateKey := env.Secret("SERVICE_PRIVATE_KEY")
		if privateKey.IsSet() {
			derived, err := servicetoken.PublicKeyPEM(privateKey.Value())
			if err != nil {
				return ServiceAuth{}, fmt.Errorf("derive SERVICE_TOKEN_PUBLIC_KEY: %w", err)
			}
			publicKey = NewSecret("SERVICE_TOKEN_PUBLIC_KEY", derived)
		}
	}
	return ServiceAuth{
		PublicKeyPEM: publicKey,
		Issuer:       env.String("SERVICE_TOKEN_ISSUER", "gateway"),
		Audience:     env.String("SERVICE_TOKEN_AUDIENCE", "halcyon-internal"),
	}, nil
}
