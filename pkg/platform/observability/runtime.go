package observability

import (
	"context"
	"log/slog"
)

// Bootstrap initializes tracing and a correlated structured logger.
func Bootstrap(ctx context.Context, serviceName, logLevel, logFormat string) (*TracerProvider, *slog.Logger, error) {
	tp, err := InitTracing(ctx, serviceName)
	if err != nil {
		return nil, nil, err
	}
	logger := NewLoggerWithTracing(logLevel, logFormat)
	slog.SetDefault(logger)
	return tp, logger, nil
}
