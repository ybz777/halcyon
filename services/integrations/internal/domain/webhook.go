package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const endpointIDPrefix = "whe_"
const deliveryIDPrefix = "whd_"

type EndpointStatus string

const (
	EndpointStatusActive   EndpointStatus = "ACTIVE"
	EndpointStatusDisabled EndpointStatus = "DISABLED"
)

type DeliveryStatus string

const (
	DeliveryPending    DeliveryStatus = "PENDING"
	DeliveryDelivering DeliveryStatus = "DELIVERING"
	DeliverySucceeded  DeliveryStatus = "SUCCEEDED"
	DeliveryFailed     DeliveryStatus = "FAILED"
	DeliveryRetryWait  DeliveryStatus = "RETRY_WAIT"
	DeliveryDisabled   DeliveryStatus = "DISABLED"
)

type EndpointID struct{ value string }
type DeliveryID struct{ value string }

func NewEndpointID() (EndpointID, error) {
	raw, err := ids.Generate(endpointIDPrefix)
	if err != nil {
		return EndpointID{}, err
	}
	return EndpointID{value: raw}, nil
}

func ParseEndpointID(raw string) (EndpointID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, endpointIDPrefix) {
		return EndpointID{}, fmt.Errorf("%w: invalid endpoint id", ErrInvalidEndpointID)
	}
	return EndpointID{value: raw}, nil
}

func (id EndpointID) String() string { return id.value }

func NewDeliveryID() (DeliveryID, error) {
	raw, err := ids.Generate(deliveryIDPrefix)
	if err != nil {
		return DeliveryID{}, err
	}
	return DeliveryID{value: raw}, nil
}

func ParseDeliveryID(raw string) (DeliveryID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, deliveryIDPrefix) {
		return DeliveryID{}, fmt.Errorf("%w: invalid delivery id", ErrInvalidDeliveryID)
	}
	return DeliveryID{value: raw}, nil
}

func (id DeliveryID) String() string { return id.value }

type WebhookEndpoint struct {
	ID               EndpointID
	OrganizationID   string
	URL              string
	Description      string
	SecretHash       string
	SecretPrefix     string
	Status           EndpointStatus
	SubscribedEvents []string
	FailureCount     int
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DisabledAt       *time.Time
}

type WebhookDelivery struct {
	ID                DeliveryID
	EndpointID        EndpointID
	OrganizationID    string
	EventID           string
	EventType         string
	Payload           []byte
	AttemptCount      int
	Status            DeliveryStatus
	AvailableAt       time.Time
	LastAttemptAt     *time.Time
	NextAttemptAt     *time.Time
	ResponseStatus    *int
	ResponseTimeMs    *int
	LastError         string
	LockedAt          *time.Time
	LockedBy          string
	CreatedAt         time.Time
	CompletedAt       *time.Time
}

func (e WebhookEndpoint) IsActive() bool { return e.Status == EndpointStatusActive }

func (e WebhookEndpoint) SubscribesTo(eventType string) bool {
	for _, subscribed := range e.SubscribedEvents {
		if subscribed == eventType || strings.HasSuffix(subscribed, ".*") && strings.HasPrefix(eventType, strings.TrimSuffix(subscribed, ".*")) {
			return true
		}
		if subscribed == eventType+".v1" || eventType == strings.TrimSuffix(subscribed, ".v1") {
			return true
		}
	}
	return false
}
