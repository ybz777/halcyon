package postgres

import (
	"context"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	jobpostgres "github.com/halcyon-io/halcyon/pkg/platform/jobs/postgres"
)

// AttemptReader adapts the platform attempt repository for paginated job queries.
type AttemptReader struct {
	inner *jobpostgres.AttemptRepository
}

// NewAttemptReader creates an AttemptReader.
func NewAttemptReader(inner *jobpostgres.AttemptRepository) *AttemptReader {
	return &AttemptReader{inner: inner}
}

// ListByJob returns a cursor-paginated attempt history.
func (r *AttemptReader) ListByJob(ctx context.Context, jobID platformjobs.ID, limit int, cursor string) ([]platformjobs.Attempt, string, error) {
	return r.inner.ListByJobPaginated(ctx, jobID, limit, cursor)
}
