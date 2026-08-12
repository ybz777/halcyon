package jobs_test

import (
	"context"
	"testing"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
)

func TestRegistryUnknownType(t *testing.T) {
	registry := platformjobs.NewRegistry()
	_, ok := registry.Get(platformjobs.TypeWebhookDelivery)
	if ok {
		t.Fatal("expected missing handler")
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := platformjobs.NewRegistry()
	err := registry.Register(platformjobs.TypeEmailDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, ok := registry.Get(platformjobs.TypeEmailDelivery)
	if !ok {
		t.Fatal("expected handler")
	}
}
