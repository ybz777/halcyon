package dto

import (
	"time"

	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
)

// CreateAPIKeyRequest creates a new API key.
type CreateAPIKeyRequest struct {
	Name             string   `json:"name"`
	ServiceAccountID string   `json:"serviceAccountId"`
	OrganizationID   string   `json:"organizationId,omitempty"`
	Scopes           []string `json:"scopes"`
	Environment      string   `json:"environment"`
	ExpiresAt        *string  `json:"expiresAt,omitempty"`
}

// APIKeyResponse is safe API key metadata.
type APIKeyResponse struct {
	ID               string   `json:"id"`
	KeyID            string   `json:"keyId"`
	ServiceAccountID string   `json:"serviceAccountId"`
	Name             string   `json:"name"`
	Prefix           string   `json:"prefix"`
	Scopes           []string `json:"scopes"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"createdAt"`
	ExpiresAt        *string  `json:"expiresAt,omitempty"`
	LastUsedAt       *string  `json:"lastUsedAt,omitempty"`
	RevokedAt        *string  `json:"revokedAt,omitempty"`
	CreatedBy        string   `json:"createdBy"`
}

// CreateAPIKeyResponse returns metadata and the plaintext key once.
type CreateAPIKeyResponse struct {
	APIKey    APIKeyResponse `json:"apiKey"`
	Plaintext string         `json:"plaintext"`
}

// RotateAPIKeyResponse returns rotated key metadata and plaintext.
type RotateAPIKeyResponse = CreateAPIKeyResponse

// ToAPIKeyResponse maps metadata to a DTO.
func ToAPIKeyResponse(meta appauth.APIKeyMetadata) APIKeyResponse {
	return APIKeyResponse{
		ID:               meta.ID,
		KeyID:            meta.KeyID,
		ServiceAccountID: meta.ServiceAccountID,
		Name:             meta.Name,
		Prefix:           meta.Prefix,
		Scopes:           meta.Scopes,
		Status:           meta.Status,
		CreatedAt:        meta.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:        formatOptionalTime(meta.ExpiresAt),
		LastUsedAt:       formatOptionalTime(meta.LastUsedAt),
		RevokedAt:        formatOptionalTime(meta.RevokedAt),
		CreatedBy:        meta.CreatedBy,
	}
}

// ToCreateAPIKeyResponse maps a creation result to a DTO.
func ToCreateAPIKeyResponse(result appauth.CreateAPIKeyResult) CreateAPIKeyResponse {
	return CreateAPIKeyResponse{
		APIKey:    ToAPIKeyResponse(result.APIKey),
		Plaintext: result.Plaintext,
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
