package apikey

import (
	"context"

	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
)

type PortAdapter struct {
	inner *Authenticator
}

func NewPortAdapter(inner *Authenticator) ports.APIKeyAuthenticator {
	if inner == nil {
		return nil
	}
	return &PortAdapter{inner: inner}
}

func (a *PortAdapter) Authenticate(ctx context.Context, apiKey string) (ports.APIKeyPrincipal, error) {
	principal, err := a.inner.Authenticate(ctx, apiKey)
	if err != nil {
		return ports.APIKeyPrincipal{}, err
	}
	return ports.APIKeyPrincipal{
		APIKeyID: principal.APIKeyID, OrganizationID: principal.OrganizationID, Scopes: principal.Scopes,
	}, nil
}
