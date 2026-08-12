package domain

import (
	"fmt"
	"strings"
)

// Status is the incident workflow state.
type Status string

const (
	StatusOpen           Status = "OPEN"
	StatusAcknowledged   Status = "ACKNOWLEDGED"
	StatusInvestigating  Status = "INVESTIGATING"
	StatusMitigated      Status = "MITIGATED"
	StatusResolved       Status = "RESOLVED"
	StatusClosed         Status = "CLOSED"
)

// ParseStatus validates a status value.
func ParseStatus(raw string) (Status, error) {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	switch Status(raw) {
	case StatusOpen, StatusAcknowledged, StatusInvestigating, StatusMitigated, StatusResolved, StatusClosed:
		return Status(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, raw)
	}
}

func (s Status) String() string { return string(s) }
func (s Status) IsTerminal() bool {
	return s == StatusClosed
}
