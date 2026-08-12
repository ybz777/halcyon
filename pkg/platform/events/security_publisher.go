package events

import (
	"context"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/eventtopics"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
)

// SecurityPublisher emits security events to Kafka for audit consumption.
type SecurityPublisher struct {
	producer kafka.Producer
	producerName string
}

// NewSecurityPublisher creates a security event publisher.
func NewSecurityPublisher(producer kafka.Producer, producerName string) *SecurityPublisher {
	if producerName == "" {
		producerName = "gateway"
	}
	return &SecurityPublisher{producer: producer, producerName: producerName}
}

// PublishInvalidToken emits a security.invalid_token event.
func (p *SecurityPublisher) PublishInvalidToken(ctx context.Context, meta SecurityMeta, reason string) error {
	return p.publish(ctx, TypeSecurityInvalidToken, "security", "invalid_token", SecurityInvalidTokenPayload{
		Reason: reason, IPAddress: meta.IPAddress, UserAgent: meta.UserAgent,
	}, meta)
}

// PublishRateLimited emits a security.rate_limited event.
func (p *SecurityPublisher) PublishRateLimited(ctx context.Context, meta SecurityMeta, routeClass string) error {
	return p.publish(ctx, TypeSecurityRateLimited, "security", "rate_limited", SecurityRateLimitedPayload{
		RouteClass: routeClass, IPAddress: meta.IPAddress,
	}, meta)
}

// PublishAuthzDenied emits a security.authz.denied event.
func (p *SecurityPublisher) PublishAuthzDenied(ctx context.Context, meta SecurityMeta, resource, permission string) error {
	return p.publish(ctx, TypeSecurityAuthzDenied, "security", "authz_denied", SecurityAuthzDeniedPayload{
		Resource: resource, Permission: permission,
	}, meta)
}

// PublishSuspiciousRequest emits a security.suspicious_request event.
func (p *SecurityPublisher) PublishSuspiciousRequest(ctx context.Context, meta SecurityMeta, reason string) error {
	return p.publish(ctx, TypeSecuritySuspiciousRequest, "security", "suspicious_request", SecuritySuspiciousRequestPayload{
		Reason: reason, IPAddress: meta.IPAddress,
	}, meta)
}

// SecurityMeta carries request correlation metadata for security events.
type SecurityMeta struct {
	RequestID     string
	CorrelationID string
	ActorID       string
	IPAddress     string
	UserAgent     string
	OrganizationID string
}

func (p *SecurityPublisher) publish(ctx context.Context, eventType, aggregateType, aggregateID string, payload any, meta SecurityMeta) error {
	if p == nil || p.producer == nil {
		return nil
	}
	orgID := meta.OrganizationID
	if orgID == "" {
		orgID = SystemOrganizationID
	}
	record, err := BuildRecord(eventType, aggregateID, aggregateType, orgID, time.Now().UTC(), payload)
	if err != nil {
		return err
	}
	envelope, err := NewEnvelope(record, Metadata{
		RequestID:     meta.RequestID,
		CorrelationID: meta.CorrelationID,
		ActorID:       meta.ActorID,
		Producer:      p.producerName,
	})
	if err != nil {
		return fmt.Errorf("build envelope: %w", err)
	}
	data, err := SerializeEnvelope(envelope)
	if err != nil {
		return fmt.Errorf("serialize envelope: %w", err)
	}
	topic := eventtopics.TopicForEventType(eventType)
	return p.producer.Publish(ctx, kafka.Message{
		Topic:         topic,
		Key:           []byte(aggregateID),
		Value:         data,
		EventID:       envelope.EventID,
		EventType:     envelope.EventType,
		EventVersion:  envelope.EventVersion,
		CorrelationID: envelope.CorrelationID,
		RequestID:     envelope.RequestID,
		Producer:      envelope.Producer,
	})
}
