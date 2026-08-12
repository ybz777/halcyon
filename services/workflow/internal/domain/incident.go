package domain

import (
	"fmt"
	"time"
)

// TransitionRecord captures a workflow state change for audit history.
type TransitionRecord struct {
	IncidentID     IncidentID
	OrganizationID OrganizationID
	FromStatus     Status
	ToStatus       Status
	FromSeverity   *Severity
	ToSeverity     *Severity
	ActorID        UserID
	Note           string
	OccurredAt     time.Time
}

// Incident is the incident management aggregate root.
type Incident struct {
	id             IncidentID
	organizationID OrganizationID
	title          Title
	description    Description
	severity       Severity
	priority       Priority
	status         Status
	source         Source
	createdAt      time.Time
	updatedAt      time.Time
	resolvedAt     *time.Time
	version        int
	events         []Event
	history        []TransitionRecord
}

// CreateIncident opens a new incident in OPEN status.
func CreateIncident(
	id IncidentID,
	organizationID OrganizationID,
	title Title,
	description Description,
	severity Severity,
	priority Priority,
	source Source,
	actorID UserID,
	now time.Time,
) (*Incident, error) {
	if id.IsZero() || organizationID.IsZero() || title.IsZero() || now.IsZero() {
		return nil, ErrInvalidTimestamp
	}

	inc := &Incident{
		id:             id,
		organizationID: organizationID,
		title:          title,
		description:    description,
		severity:       severity,
		priority:       priority,
		status:         StatusOpen,
		source:         source,
		createdAt:      now.UTC(),
		updatedAt:      now.UTC(),
		version:        1,
	}
	inc.recordHistory(TransitionRecord{
		IncidentID:     id,
		OrganizationID: organizationID,
		FromStatus:     "",
		ToStatus:       StatusOpen,
		ActorID:        actorID,
		OccurredAt:     now.UTC(),
	})
	inc.record(NewIncidentCreated(id, organizationID, title, severity, priority, source, now.UTC()))
	return inc, nil
}

// ReconstituteIncident rebuilds an incident from persistence.
func ReconstituteIncident(
	id IncidentID,
	organizationID OrganizationID,
	title Title,
	description Description,
	severity Severity,
	priority Priority,
	status Status,
	source Source,
	createdAt, updatedAt time.Time,
	resolvedAt *time.Time,
	version int,
) (*Incident, error) {
	if id.IsZero() || organizationID.IsZero() || title.IsZero() {
		return nil, ErrInvalidIncidentID
	}
	return &Incident{
		id:             id,
		organizationID: organizationID,
		title:          title,
		description:    description,
		severity:       severity,
		priority:       priority,
		status:         status,
		source:         source,
		createdAt:      createdAt.UTC(),
		updatedAt:      updatedAt.UTC(),
		resolvedAt:     resolvedAt,
		version:        version,
	}, nil
}

// TransitionTo moves the incident to a new status when allowed.
func (i *Incident) TransitionTo(target Status, actorID UserID, now time.Time) error {
	if i.status.IsTerminal() {
		return ErrIncidentClosed
	}
	if !CanTransition(i.status, target) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, i.status, target)
	}
	return i.applyTransition(i.status, target, actorID, now)
}

// Resolve transitions the incident to RESOLVED.
func (i *Incident) Resolve(actorID UserID, now time.Time) error {
	return i.TransitionTo(StatusResolved, actorID, now)
}

// Close transitions the incident to CLOSED.
func (i *Incident) Close(actorID UserID, now time.Time) error {
	return i.TransitionTo(StatusClosed, actorID, now)
}

// UpdateSeverity changes severity and emits escalation when increased.
func (i *Incident) UpdateSeverity(next Severity, actorID UserID, now time.Time) error {
	if i.status.IsTerminal() {
		return ErrIncidentClosed
	}
	if next == i.severity {
		return nil
	}

	prev := i.severity
	i.severity = next
	i.updatedAt = now.UTC()
	i.version++

	i.recordHistory(TransitionRecord{
		IncidentID:     i.id,
		OrganizationID: i.organizationID,
		FromStatus:     i.status,
		ToStatus:       i.status,
		FromSeverity:   &prev,
		ToSeverity:     &next,
		ActorID:        actorID,
		OccurredAt:     now.UTC(),
	})

	if next.IsHigherThan(prev) {
		i.record(NewIncidentEscalated(i.id, i.organizationID, prev, next, now.UTC()))
	}
	return nil
}

func (i *Incident) applyTransition(from, to Status, actorID UserID, now time.Time) error {
	i.status = to
	i.updatedAt = now.UTC()
	i.version++

	if to == StatusResolved {
		resolved := now.UTC()
		i.resolvedAt = &resolved
	}

	i.recordHistory(TransitionRecord{
		IncidentID:     i.id,
		OrganizationID: i.organizationID,
		FromStatus:     from,
		ToStatus:       to,
		ActorID:        actorID,
		OccurredAt:     now.UTC(),
	})

	i.record(NewStateChanged(i.id, i.organizationID, from, to, now.UTC()))

	switch to {
	case StatusAcknowledged:
		i.record(NewIncidentAcknowledged(i.id, i.organizationID, now.UTC()))
	case StatusResolved:
		i.record(NewIncidentResolved(i.id, i.organizationID, now.UTC()))
	case StatusClosed:
		i.record(NewIncidentClosed(i.id, i.organizationID, now.UTC()))
	}
	return nil
}

func (i *Incident) record(event Event)              { i.events = append(i.events, event) }
func (i *Incident) recordHistory(record TransitionRecord) { i.history = append(i.history, record) }

func (i *Incident) ID() IncidentID                   { return i.id }
func (i *Incident) OrganizationID() OrganizationID { return i.organizationID }
func (i *Incident) Title() Title                     { return i.title }
func (i *Incident) Description() Description         { return i.description }
func (i *Incident) Severity() Severity               { return i.severity }
func (i *Incident) Priority() Priority               { return i.priority }
func (i *Incident) Status() Status                   { return i.status }
func (i *Incident) Source() Source                   { return i.source }
func (i *Incident) CreatedAt() time.Time             { return i.createdAt }
func (i *Incident) UpdatedAt() time.Time             { return i.updatedAt }
func (i *Incident) ResolvedAt() *time.Time           { return i.resolvedAt }
func (i *Incident) Version() int                     { return i.version }
func (i *Incident) UncommittedEvents() []Event       { return append([]Event(nil), i.events...) }
func (i *Incident) PendingHistory() []TransitionRecord {
	return append([]TransitionRecord(nil), i.history...)
}
func (i *Incident) ClearUncommittedEvents() { i.events = nil }
func (i *Incident) ClearPendingHistory()    { i.history = nil }
