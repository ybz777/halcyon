package domain

import "time"

// Organization is the tenancy aggregate root.
type Organization struct {
	id        OrganizationID
	name      OrganizationName
	slug      Slug
	status    OrganizationStatus
	createdAt time.Time
	updatedAt time.Time
	version   int
	events    []Event
}

// CreateOrganization creates a new active organization aggregate.
func CreateOrganization(id OrganizationID, name OrganizationName, slug Slug, now time.Time) (*Organization, error) {
	if id.IsZero() || name.IsZero() || slug.IsZero() || now.IsZero() {
		return nil, ErrInvalidTimestamp
	}

	now = now.UTC()
	org := &Organization{
		id:        id,
		name:      name,
		slug:      slug,
		status:    OrganizationStatusActive,
		createdAt: now,
		updatedAt: now,
		version:   1,
	}
	org.record(NewOrganizationCreated(id, name, slug, now))
	return org, nil
}

// ReconstituteOrganization rebuilds an organization from persistence.
func ReconstituteOrganization(
	id OrganizationID,
	name OrganizationName,
	slug Slug,
	status OrganizationStatus,
	createdAt, updatedAt time.Time,
	version int,
) (*Organization, error) {
	if id.IsZero() || name.IsZero() || slug.IsZero() {
		return nil, ErrInvalidOrganizationID
	}
	if !isValidOrganizationStatus(status) {
		return nil, ErrInvalidOrganizationStatus
	}
	return &Organization{
		id:        id,
		name:      name,
		slug:      slug,
		status:    status,
		createdAt: createdAt.UTC(),
		updatedAt: updatedAt.UTC(),
		version:   version,
	}, nil
}

// UpdateName changes the organization display name.
func (o *Organization) UpdateName(name OrganizationName, now time.Time) error {
	if err := o.ensureOperational(); err != nil {
		return err
	}
	if name.IsZero() {
		return ErrInvalidOrganizationName
	}
	o.name = name
	o.updatedAt = now.UTC()
	o.version++
	o.record(NewOrganizationUpdated(o.id, o.name, now.UTC()))
	return nil
}

// Suspend marks the organization suspended.
func (o *Organization) Suspend(now time.Time) error {
	if o.status == OrganizationStatusDeleted {
		return ErrOrganizationDeleted
	}
	if o.status == OrganizationStatusSuspended {
		return nil
	}
	o.status = OrganizationStatusSuspended
	o.updatedAt = now.UTC()
	o.version++
	return nil
}

// SoftDelete marks the organization deleted.
func (o *Organization) SoftDelete(now time.Time) error {
	if o.status == OrganizationStatusDeleted {
		return nil
	}
	o.status = OrganizationStatusDeleted
	o.updatedAt = now.UTC()
	o.version++
	o.record(NewOrganizationDeleted(o.id, now.UTC()))
	return nil
}

func (o *Organization) ensureOperational() error {
	switch o.status {
	case OrganizationStatusActive:
		return nil
	case OrganizationStatusSuspended:
		return ErrOrganizationSuspended
	case OrganizationStatusDeleted:
		return ErrOrganizationDeleted
	default:
		return ErrOrganizationSuspended
	}
}

func (o *Organization) record(event Event) { o.events = append(o.events, event) }

func (o *Organization) ID() OrganizationID         { return o.id }
func (o *Organization) Name() OrganizationName   { return o.name }
func (o *Organization) Slug() Slug               { return o.slug }
func (o *Organization) Status() OrganizationStatus { return o.status }
func (o *Organization) CreatedAt() time.Time     { return o.createdAt }
func (o *Organization) UpdatedAt() time.Time     { return o.updatedAt }
func (o *Organization) Version() int             { return o.version }
func (o *Organization) UncommittedEvents() []Event {
	if len(o.events) == 0 {
		return nil
	}
	out := make([]Event, len(o.events))
	copy(out, o.events)
	return out
}
func (o *Organization) ClearUncommittedEvents() { o.events = nil }
