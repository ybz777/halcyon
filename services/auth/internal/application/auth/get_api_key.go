package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// GetAPIKeyQuery loads API key metadata by ID.
type GetAPIKeyQuery struct {
	APIKeyID domain.APIKeyID
}

// GetAPIKeyHandler returns API key metadata without secrets.
type GetAPIKeyHandler struct {
	apiKeys ports.APIKeyRepository
}

// NewGetAPIKeyHandler creates a GetAPIKeyHandler.
func NewGetAPIKeyHandler(apiKeys ports.APIKeyRepository) *GetAPIKeyHandler {
	return &GetAPIKeyHandler{apiKeys: apiKeys}
}

// Handle returns API key metadata.
func (h *GetAPIKeyHandler) Handle(ctx context.Context, query GetAPIKeyQuery) (APIKeyMetadata, error) {
	key, err := h.apiKeys.FindByID(ctx, query.APIKeyID)
	if err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			return APIKeyMetadata{}, domain.ErrAPIKeyNotFound
		}
		return APIKeyMetadata{}, fmt.Errorf("find api key: %w", err)
	}
	return toAPIKeyMetadata(key), nil
}
