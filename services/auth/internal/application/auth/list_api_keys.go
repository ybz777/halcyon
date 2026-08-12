package auth

import (
	"context"
	"fmt"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// ListAPIKeysQuery lists API keys for a service account.
type ListAPIKeysQuery struct {
	ServiceAccountID domain.ServiceAccountID
	Limit            int
	Cursor           string
}

// ListAPIKeysHandler lists API key metadata.
type ListAPIKeysHandler struct {
	apiKeys ports.APIKeyRepository
}

// NewListAPIKeysHandler creates a ListAPIKeysHandler.
func NewListAPIKeysHandler(apiKeys ports.APIKeyRepository) *ListAPIKeysHandler {
	return &ListAPIKeysHandler{apiKeys: apiKeys}
}

// Handle returns API key metadata without secrets.
func (h *ListAPIKeysHandler) Handle(ctx context.Context, query ListAPIKeysQuery) ([]APIKeyMetadata, string, error) {
	keys, cursor, err := h.apiKeys.ListByServiceAccount(ctx, query.ServiceAccountID, query.Limit, query.Cursor)
	if err != nil {
		return nil, "", fmt.Errorf("list api keys: %w", err)
	}

	result := make([]APIKeyMetadata, 0, len(keys))
	for _, key := range keys {
		result = append(result, toAPIKeyMetadata(key))
	}
	return result, cursor, nil
}
