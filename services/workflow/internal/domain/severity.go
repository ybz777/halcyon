package domain

import (
	"fmt"
	"strings"
)

// Severity classifies incident impact.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

var severityRank = map[Severity]int{
	SeverityInfo:     0,
	SeverityLow:      1,
	SeverityMedium:   2,
	SeverityHigh:     3,
	SeverityCritical: 4,
}

// ParseSeverity validates a severity value.
func ParseSeverity(raw string) (Severity, error) {
	raw = strings.TrimSpace(strings.ToUpper(raw))
	switch Severity(raw) {
	case SeverityInfo, SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return Severity(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidSeverity, raw)
	}
}

func (s Severity) String() string { return string(s) }

// IsHigherThan reports whether s is more severe than other.
func (s Severity) IsHigherThan(other Severity) bool {
	return severityRank[s] > severityRank[other]
}
