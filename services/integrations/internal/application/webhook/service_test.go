package webhook_test

import (
	"context"
	"testing"

	platformwebhook "github.com/halcyon-io/halcyon/pkg/platform/webhook"
	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
)

type stubEndpoints struct {
	count int
}

func (s *stubEndpoints) Create(context.Context, domain.WebhookEndpoint, string) error { return nil }
func (s *stubEndpoints) Update(context.Context, domain.WebhookEndpoint) error         { return nil }
func (s *stubEndpoints) FindByID(context.Context, string, domain.EndpointID) (domain.WebhookEndpoint, error) {
	return domain.WebhookEndpoint{}, domain.ErrEndpointNotFound
}
func (s *stubEndpoints) FindByIDWithSecret(context.Context, string, domain.EndpointID) (domain.WebhookEndpoint, string, error) {
	return domain.WebhookEndpoint{}, "", domain.ErrEndpointNotFound
}
func (s *stubEndpoints) ListByOrganization(context.Context, string, int, string) ([]domain.WebhookEndpoint, string, error) {
	return nil, "", nil
}
func (s *stubEndpoints) CountByOrganization(context.Context, string) (int, error) { return s.count, nil }

type stubDeliveries struct{}

func (stubDeliveries) CreateIdempotent(context.Context, domain.WebhookDelivery) (bool, error) { return false, nil }
func (stubDeliveries) ListByEndpoint(context.Context, string, domain.EndpointID, int, string) ([]domain.WebhookDelivery, string, error) {
	return nil, "", nil
}
func (stubDeliveries) FindByID(context.Context, string, domain.DeliveryID) (domain.WebhookDelivery, error) {
	return domain.WebhookDelivery{}, domain.ErrDeliveryNotFound
}

func TestCreateEndpointRejectsPrivateURL(t *testing.T) {
	cipher, err := crypto.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	svc := appwebhook.NewService(&stubEndpoints{}, stubDeliveries{}, cipher, false, 0)
	_, err = svc.CreateEndpoint(context.Background(), appwebhook.CreateEndpointInput{
		Principal: appwebhook.Principal{Permissions: map[string]struct{}{
			domain.PermissionIntegrationsWrite: {},
		}},
		OrganizationID: "org_test",
		URL:            "https://127.0.0.1/hook",
		Events:         []string{"workflow.execution.succeeded.v1"},
	})
	if err == nil {
		t.Fatal("expected SSRF rejection")
	}
}

func TestCreateEndpointRequiresPermission(t *testing.T) {
	cipher, _ := crypto.NewSecretCipher("0123456789abcdef0123456789abcdef")
	svc := appwebhook.NewService(&stubEndpoints{}, stubDeliveries{}, cipher, true, 0)
	_, err := svc.CreateEndpoint(context.Background(), appwebhook.CreateEndpointInput{
		Principal:      appwebhook.Principal{},
		OrganizationID: "org_test",
		URL:            "http://example.com/hook",
		Events:         []string{"workflow.execution.succeeded.v1"},
	})
	if err != domain.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestWebhookSecretGeneration(t *testing.T) {
	secret, prefix, hash, err := platformwebhook.GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" || prefix == "" || hash == "" {
		t.Fatal("expected secret material")
	}
	if !platformwebhook.VerifySecret(secret, hash) {
		t.Fatal("hash verification failed")
	}
}
