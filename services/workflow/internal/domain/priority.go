package domain

import (
	"fmt"
	"strings"
)

// Priority classifies incident urgency.
type Priority string

const (
	PriorityLow      Priority = "LOW"
	PriorityMedium   Priority = "MEDIUM"
	PriorityHigh     Priority = "HIGH"
	PriorityCritical Priority = "CRITICAL"
)

// ParsePriority validates a priority value.
func ParsePriority(raw string) (Priority, error) {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	switch Priority(raw) {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return Priority(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidPriority, raw)
	}
}

func (p Priority) String() string { return string(p) }
