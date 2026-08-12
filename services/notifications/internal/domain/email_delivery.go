package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const emailDeliveryIDPrefix = "eml_"

type EmailDeliveryID struct{ value string }

func NewEmailDeliveryID() (EmailDeliveryID, error) {
	raw, err := ids.Generate(emailDeliveryIDPrefix)
	if err != nil {
		return EmailDeliveryID{}, err
	}
	return EmailDeliveryID{value: raw}, nil
}

func ParseEmailDeliveryID(raw string) (EmailDeliveryID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, emailDeliveryIDPrefix) {
		return EmailDeliveryID{}, fmt.Errorf("%w: invalid id", ErrInvalidEmailDeliveryID)
	}
	return EmailDeliveryID{value: raw}, nil
}

func (id EmailDeliveryID) String() string { return id.value }

type EmailDelivery struct {
	ID                EmailDeliveryID
	NotificationID    NotificationID
	UserID            string
	Email             string
	Template          string
	Status            EmailDeliveryStatus
	AttemptCount      int
	AvailableAt       time.Time
	LastAttemptAt     *time.Time
	NextAttemptAt     *time.Time
	Provider          string
	ProviderMessageID string
	LastError         string
	LockedAt          *time.Time
	LockedBy          string
	CreatedAt         time.Time
	CompletedAt       *time.Time
}

type NotificationPreference struct {
	UserID           string
	NotificationType NotificationType
	Channel          Channel
	Enabled          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
