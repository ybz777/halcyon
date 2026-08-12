package securityevents

import (
	"context"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
)

// Publisher publishes gateway security events to Kafka.
type Publisher struct {
	inner *platformevents.SecurityPublisher
}

// NewPublisher creates a security event publisher. Returns nil when producer is nil.
func NewPublisher(producer kafka.Producer) ports.SecurityEventPublisher {
	if producer == nil {
		return noopPublisher{}
	}
	return &Publisher{inner: platformevents.NewSecurityPublisher(producer, "gateway")}
}

type noopPublisher struct{}

func (noopPublisher) PublishInvalidToken(context.Context, string, string, string, string, string) error {
	return nil
}

func (noopPublisher) PublishRateLimited(context.Context, string, string, string, string) error {
	return nil
}

func (p *Publisher) PublishInvalidToken(ctx context.Context, requestID, correlationID, reason, ip, userAgent string) error {
	return p.inner.PublishInvalidToken(ctx, platformevents.SecurityMeta{
		RequestID: requestID, CorrelationID: correlationID, IPAddress: ip, UserAgent: userAgent,
	}, reason)
}

func (p *Publisher) PublishRateLimited(ctx context.Context, requestID, correlationID, routeClass, ip string) error {
	return p.inner.PublishRateLimited(ctx, platformevents.SecurityMeta{
		RequestID: requestID, CorrelationID: correlationID, IPAddress: ip,
	}, routeClass)
}
