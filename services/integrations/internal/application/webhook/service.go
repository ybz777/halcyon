package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	platformwebhook "github.com/halcyon-io/halcyon/pkg/platform/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
)

type EndpointRepository interface {
	Create(ctx context.Context, endpoint domain.WebhookEndpoint, secretCiphertext string) error
	Update(ctx context.Context, endpoint domain.WebhookEndpoint) error
	FindByID(ctx context.Context, orgID string, id domain.EndpointID) (domain.WebhookEndpoint, error)
	FindByIDWithSecret(ctx context.Context, orgID string, id domain.EndpointID) (domain.WebhookEndpoint, string, error)
	ListByOrganization(ctx context.Context, orgID string, limit int, cursor string) ([]domain.WebhookEndpoint, string, error)
	CountByOrganization(ctx context.Context, orgID string) (int, error)
}

type DeliveryRepository interface {
	CreateIdempotent(ctx context.Context, delivery domain.WebhookDelivery) (duplicate bool, err error)
	ListByEndpoint(ctx context.Context, orgID string, endpointID domain.EndpointID, limit int, cursor string) ([]domain.WebhookDelivery, string, error)
	FindByID(ctx context.Context, orgID string, id domain.DeliveryID) (domain.WebhookDelivery, error)
}

type Principal struct {
	UserID      string
	Permissions map[string]struct{}
}

func (p Principal) HasPermission(perm string) bool {
	_, ok := p.Permissions[perm]
	return ok
}

type Service struct {
	endpoints EndpointRepository
	deliveries DeliveryRepository
	cipher    *crypto.SecretCipher
	allowHTTP bool
	timeout   time.Duration
}

func NewService(endpoints EndpointRepository, deliveries DeliveryRepository, cipher *crypto.SecretCipher, allowHTTP bool, timeout time.Duration) *Service {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Service{endpoints: endpoints, deliveries: deliveries, cipher: cipher, allowHTTP: allowHTTP, timeout: timeout}
}

type CreateEndpointInput struct {
	Principal      Principal
	OrganizationID string
	URL            string
	Description    string
	Events         []string
}

type CreateEndpointResult struct {
	Endpoint     domain.WebhookEndpoint
	PlainSecret  string
}

func (s *Service) CreateEndpoint(ctx context.Context, input CreateEndpointInput) (CreateEndpointResult, error) {
	if !input.Principal.HasPermission(domain.PermissionIntegrationsWrite) {
		return CreateEndpointResult{}, domain.ErrForbidden
	}
	if err := platformwebhook.ValidateDestinationURL(input.URL, platformwebhook.URLPolicy{AllowHTTP: s.allowHTTP}); err != nil {
		return CreateEndpointResult{}, domain.ErrInvalidURL
	}
	if len(input.Events) == 0 || len(input.Events) > domain.MaxSubscriptions {
		return CreateEndpointResult{}, domain.ErrInvalidSubscription
	}
	count, err := s.endpoints.CountByOrganization(ctx, input.OrganizationID)
	if err != nil {
		return CreateEndpointResult{}, err
	}
	if count >= domain.MaxEndpointsPerOrg {
		return CreateEndpointResult{}, fmt.Errorf("endpoint limit reached")
	}

	secret, prefix, hash, err := platformwebhook.GenerateSecret()
	if err != nil {
		return CreateEndpointResult{}, err
	}
	ciphertext, err := s.cipher.Encrypt(secret)
	if err != nil {
		return CreateEndpointResult{}, err
	}
	id, err := domain.NewEndpointID()
	if err != nil {
		return CreateEndpointResult{}, err
	}
	now := time.Now().UTC()
	endpoint := domain.WebhookEndpoint{
		ID: id, OrganizationID: input.OrganizationID, URL: input.URL,
		Description: input.Description, SecretHash: hash, SecretPrefix: prefix,
		Status: domain.EndpointStatusActive, SubscribedEvents: input.Events,
		CreatedBy: input.Principal.UserID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.endpoints.Create(ctx, endpoint, ciphertext); err != nil {
		return CreateEndpointResult{}, err
	}
	return CreateEndpointResult{Endpoint: endpoint, PlainSecret: secret}, nil
}

func (s *Service) ListEndpoints(ctx context.Context, principal Principal, orgID string, limit int, cursor string) ([]domain.WebhookEndpoint, string, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsRead) {
		return nil, "", domain.ErrForbidden
	}
	return s.endpoints.ListByOrganization(ctx, orgID, limit, cursor)
}

func (s *Service) GetEndpoint(ctx context.Context, principal Principal, orgID string, id domain.EndpointID) (domain.WebhookEndpoint, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsRead) {
		return domain.WebhookEndpoint{}, domain.ErrForbidden
	}
	return s.endpoints.FindByID(ctx, orgID, id)
}

func (s *Service) SetEndpointStatus(ctx context.Context, principal Principal, orgID string, id domain.EndpointID, active bool) (domain.WebhookEndpoint, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsWrite) {
		return domain.WebhookEndpoint{}, domain.ErrForbidden
	}
	endpoint, err := s.endpoints.FindByID(ctx, orgID, id)
	if err != nil {
		return domain.WebhookEndpoint{}, err
	}
	now := time.Now().UTC()
	endpoint.UpdatedAt = now
	if active {
		endpoint.Status = domain.EndpointStatusActive
		endpoint.DisabledAt = nil
	} else {
		endpoint.Status = domain.EndpointStatusDisabled
		endpoint.DisabledAt = &now
	}
	if err := s.endpoints.Update(ctx, endpoint); err != nil {
		return domain.WebhookEndpoint{}, err
	}
	return endpoint, nil
}

func (s *Service) PatchEndpoint(ctx context.Context, principal Principal, orgID string, id domain.EndpointID, url, description string, events []string) (domain.WebhookEndpoint, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsWrite) {
		return domain.WebhookEndpoint{}, domain.ErrForbidden
	}
	endpoint, err := s.endpoints.FindByID(ctx, orgID, id)
	if err != nil {
		return domain.WebhookEndpoint{}, err
	}
	if url != "" {
		if err := platformwebhook.ValidateDestinationURL(url, platformwebhook.URLPolicy{AllowHTTP: s.allowHTTP}); err != nil {
			return domain.WebhookEndpoint{}, domain.ErrInvalidURL
		}
		endpoint.URL = url
	}
	if description != "" {
		endpoint.Description = description
	}
	if len(events) > 0 {
		if len(events) > domain.MaxSubscriptions {
			return domain.WebhookEndpoint{}, domain.ErrInvalidSubscription
		}
		endpoint.SubscribedEvents = events
	}
	endpoint.UpdatedAt = time.Now().UTC()
	if err := s.endpoints.Update(ctx, endpoint); err != nil {
		return domain.WebhookEndpoint{}, err
	}
	return endpoint, nil
}

func (s *Service) ListDeliveries(ctx context.Context, principal Principal, orgID string, endpointID domain.EndpointID, limit int, cursor string) ([]domain.WebhookDelivery, string, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsRead) {
		return nil, "", domain.ErrForbidden
	}
	return s.deliveries.ListByEndpoint(ctx, orgID, endpointID, limit, cursor)
}

func (s *Service) GetDelivery(ctx context.Context, principal Principal, orgID string, id domain.DeliveryID) (domain.WebhookDelivery, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsRead) {
		return domain.WebhookDelivery{}, domain.ErrForbidden
	}
	return s.deliveries.FindByID(ctx, orgID, id)
}

func (s *Service) BuildTestPayload(endpointID domain.EndpointID) []byte {
	payload, _ := json.Marshal(map[string]any{
		"id":         "evt_test_webhook",
		"type":       "integrations.webhook.test.v1",
		"endpointId": endpointID.String(),
		"occurredAt": time.Now().UTC().Format(time.RFC3339),
	})
	return payload
}

type TestEndpointResult struct {
	EventID        string
	EventType      string
	StatusCode     int
	ResponseTimeMs int64
	Success        bool
	ErrorSummary   string
}

func (s *Service) TestEndpoint(ctx context.Context, principal Principal, orgID string, id domain.EndpointID) (TestEndpointResult, error) {
	if !principal.HasPermission(domain.PermissionIntegrationsWrite) {
		return TestEndpointResult{}, domain.ErrForbidden
	}
	endpoint, ciphertext, err := s.endpoints.FindByIDWithSecret(ctx, orgID, id)
	if err != nil {
		return TestEndpointResult{}, err
	}
	if !endpoint.IsActive() {
		return TestEndpointResult{}, domain.ErrEndpointDisabled
	}
	secret, err := s.cipher.Decrypt(ciphertext)
	if err != nil {
		return TestEndpointResult{}, err
	}
	eventID := "evt_test_" + id.String()
	eventType := "integrations.webhook.test.v1"
	payload := s.BuildTestPayload(id)
	client := platformwebhook.NewClient(platformwebhook.ClientConfig{Timeout: s.timeout, AllowHTTP: s.allowHTTP})
	result := client.Deliver(ctx, endpoint.URL, secret, eventID, eventType, payload)

	now := time.Now().UTC()
	status := domain.DeliveryFailed
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		status = domain.DeliverySucceeded
	}
	deliveryID, _ := domain.NewDeliveryID()
	rt := int(result.ResponseTimeMs)
	_, _ = s.deliveries.CreateIdempotent(ctx, domain.WebhookDelivery{
		ID: deliveryID, EndpointID: id, OrganizationID: orgID, EventID: eventID, EventType: eventType,
		Payload: payload, AttemptCount: 1, Status: status, AvailableAt: now, CreatedAt: now,
		LastAttemptAt: &now, ResponseStatus: &result.StatusCode, ResponseTimeMs: &rt, LastError: result.ErrorSummary,
		CompletedAt: &now,
	})

	return TestEndpointResult{
		EventID: eventID, EventType: eventType, StatusCode: result.StatusCode,
		ResponseTimeMs: result.ResponseTimeMs, Success: result.StatusCode >= 200 && result.StatusCode < 300,
		ErrorSummary: result.ErrorSummary,
	}, nil
}
