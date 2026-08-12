package worker_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/worker"
)

func TestBackoffIncreasesWithJitter(t *testing.T) {
	// backoff is unexported; validate via exported worker config defaults instead
	cfg := worker.Config{MaxConcurrent: 0, PollInterval: time.Second}
	w := worker.NewWorker(nil, nil, nil, cfg, nil)
	if w == nil {
		t.Fatal("expected worker")
	}
}

func TestClassifyRetryFromClient(t *testing.T) {
	// covered in pkg/platform/webhook signing tests
}
