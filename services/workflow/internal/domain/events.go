package domain

import "time"

// Event is a domain event emitted by the incident aggregate.
type Event interface {
	EventName() string
	OccurredAt() time.Time
}

type baseEvent struct {
	name string
	at   time.Time
}

func (e baseEvent) EventName() string     { return e.name }
func (e baseEvent) OccurredAt() time.Time { return e.at }

// IncidentCreated is emitted when a new incident is opened.
type IncidentCreated struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
	Title          Title
	Severity       Severity
	Priority       Priority
	Source         Source
}

func NewIncidentCreated(id IncidentID, orgID OrganizationID, title Title, severity Severity, priority Priority, source Source, at time.Time) IncidentCreated {
	return IncidentCreated{
		baseEvent:      baseEvent{name: "incidents.incident.created", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
		Title:          title,
		Severity:       severity,
		Priority:       priority,
		Source:         source,
	}
}

// IncidentAcknowledged is emitted when an incident is acknowledged.
type IncidentAcknowledged struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
}

func NewIncidentAcknowledged(id IncidentID, orgID OrganizationID, at time.Time) IncidentAcknowledged {
	return IncidentAcknowledged{
		baseEvent:      baseEvent{name: "incidents.incident.acknowledged", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
	}
}

// IncidentEscalated is emitted when severity increases.
type IncidentEscalated struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
	FromSeverity   Severity
	ToSeverity     Severity
}

func NewIncidentEscalated(id IncidentID, orgID OrganizationID, from, to Severity, at time.Time) IncidentEscalated {
	return IncidentEscalated{
		baseEvent:      baseEvent{name: "incidents.incident.escalated", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
		FromSeverity:   from,
		ToSeverity:     to,
	}
}

// IncidentResolved is emitted when an incident is resolved.
type IncidentResolved struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
}

func NewIncidentResolved(id IncidentID, orgID OrganizationID, at time.Time) IncidentResolved {
	return IncidentResolved{
		baseEvent:      baseEvent{name: "incidents.incident.resolved", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
	}
}

// IncidentClosed is emitted when an incident is closed.
type IncidentClosed struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
}

func NewIncidentClosed(id IncidentID, orgID OrganizationID, at time.Time) IncidentClosed {
	return IncidentClosed{
		baseEvent:      baseEvent{name: "incidents.incident.closed", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
	}
}

// StateChanged is emitted on every status transition.
type StateChanged struct {
	baseEvent
	IncidentID     IncidentID
	OrganizationID OrganizationID
	FromStatus     Status
	ToStatus       Status
}

func NewStateChanged(id IncidentID, orgID OrganizationID, from, to Status, at time.Time) StateChanged {
	return StateChanged{
		baseEvent:      baseEvent{name: "incidents.incident.state_changed", at: at},
		IncidentID:     id,
		OrganizationID: orgID,
		FromStatus:     from,
		ToStatus:       to,
	}
}
