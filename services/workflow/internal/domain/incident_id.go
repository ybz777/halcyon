package domain

import (
	"fmt"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const incidentIDPrefix = "inc_"

// IncidentID is an immutable incident identifier.
type IncidentID struct {
	value string
}

// NewIncidentID generates a new incident ID.
func NewIncidentID() (IncidentID, error) {
	raw, err := ids.Generate(incidentIDPrefix)
	if err != nil {
		return IncidentID{}, fmt.Errorf("generate incident id: %w", err)
	}
	return ParseIncidentID(raw)
}

// ParseIncidentID validates a persisted incident ID.
func ParseIncidentID(raw string) (IncidentID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, incidentIDPrefix) {
		return IncidentID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidIncidentID, incidentIDPrefix)
	}
	suffix := strings.TrimPrefix(raw, incidentIDPrefix)
	if len(suffix) < 16 {
		return IncidentID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidIncidentID)
	}
	return IncidentID{value: raw}, nil
}

func (id IncidentID) String() string { return id.value }
func (id IncidentID) IsZero() bool   { return id.value == "" }
