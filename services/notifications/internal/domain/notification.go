package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const notificationIDPrefix = "ntf_"

type NotificationID struct{ value string }

func NewNotificationID() (NotificationID, error) {
	raw, err := ids.Generate(notificationIDPrefix)
	if err != nil {
		return NotificationID{}, err
	}
	return NotificationID{value: raw}, nil
}

func ParseNotificationID(raw string) (NotificationID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, notificationIDPrefix) {
		return NotificationID{}, fmt.Errorf("%w: invalid id", ErrInvalidNotificationID)
	}
	return NotificationID{value: raw}, nil
}

func (id NotificationID) String() string { return id.value }

type Notification struct {
	ID               NotificationID
	OrganizationID   string
	UserID           string
	EventID          string
	EventType        string
	NotificationType NotificationType
	Title            string
	Body             string
	ResourceType     string
	ResourceID       string
	Metadata         map[string]string
	ReadAt           *time.Time
	CreatedAt        time.Time
}

func (n Notification) IsUnread() bool { return n.ReadAt == nil }
