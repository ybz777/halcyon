package domain

import "time"

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User is the aggregate root for identity credentials in the auth bounded context.
type User struct {
	id           UserID
	email        Email
	passwordHash PasswordHash
	status       UserStatus
	createdAt    time.Time
	updatedAt    time.Time
	version      int
	events       []Event
}

// RegisterUser creates a new active user and records a UserRegistered domain event.
func RegisterUser(id UserID, email Email, passwordHash PasswordHash, now time.Time) (*User, error) {
	if id.IsZero() {
		return nil, ErrInvalidUserID
	}
	if email.IsZero() {
		return nil, ErrInvalidEmail
	}
	if passwordHash.IsZero() {
		return nil, ErrInvalidPasswordHash
	}
	if now.IsZero() {
		return nil, ErrInvalidTimestamp
	}

	user := &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		status:       UserStatusActive,
		createdAt:    now.UTC(),
		updatedAt:    now.UTC(),
		version:      1,
	}

	user.record(NewUserRegistered(id, email, now.UTC()))

	return user, nil
}

// ReconstituteUser rebuilds a User from persisted state without emitting events.
func ReconstituteUser(
	id UserID,
	email Email,
	passwordHash PasswordHash,
	status UserStatus,
	createdAt, updatedAt time.Time,
	version int,
) (*User, error) {
	if id.IsZero() {
		return nil, ErrInvalidUserID
	}
	if email.IsZero() {
		return nil, ErrInvalidEmail
	}
	if passwordHash.IsZero() {
		return nil, ErrInvalidPasswordHash
	}
	if !isValidUserStatus(status) {
		return nil, ErrInvalidUserStatus
	}
	if createdAt.IsZero() || updatedAt.IsZero() {
		return nil, ErrInvalidTimestamp
	}
	if version < 1 {
		return nil, ErrInvalidVersion
	}

	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		status:       status,
		createdAt:    createdAt.UTC(),
		updatedAt:    updatedAt.UTC(),
		version:      version,
	}, nil
}

// ChangeEmail updates the user's email when the account is active.
func (u *User) ChangeEmail(newEmail Email, now time.Time) error {
	if err := u.requireActive(); err != nil {
		return err
	}
	if newEmail.IsZero() {
		return ErrInvalidEmail
	}
	if u.email.Equal(newEmail) {
		return ErrEmailUnchanged
	}
	if now.IsZero() {
		return ErrInvalidTimestamp
	}

	oldEmail := u.email
	u.email = newEmail
	u.updatedAt = now.UTC()
	u.version++

	u.record(NewUserEmailChanged(u.id, oldEmail, newEmail, now.UTC()))

	return nil
}

// Disable deactivates the user account.
func (u *User) Disable(now time.Time) error {
	if u.status == UserStatusDisabled {
		return nil
	}
	if now.IsZero() {
		return ErrInvalidTimestamp
	}

	u.status = UserStatusDisabled
	u.updatedAt = now.UTC()
	u.version++

	u.record(NewUserDisabled(u.id, now.UTC()))

	return nil
}

// Enable reactivates a disabled user account.
func (u *User) Enable(now time.Time) error {
	if u.status == UserStatusActive {
		return nil
	}
	if now.IsZero() {
		return ErrInvalidTimestamp
	}

	u.status = UserStatusActive
	u.updatedAt = now.UTC()
	u.version++

	u.record(NewUserEnabled(u.id, now.UTC()))

	return nil
}

// RequireActive returns an error if the user cannot authenticate.
func (u *User) RequireActive() error {
	return u.requireActive()
}

func (u *User) requireActive() error {
	if u.status == UserStatusDisabled {
		return ErrUserDisabled
	}
	return nil
}

func (u *User) record(event Event) {
	u.events = append(u.events, event)
}

// ID returns the user identifier.
func (u *User) ID() UserID { return u.id }

// Email returns the user's email address.
func (u *User) Email() Email { return u.email }

// PasswordHash returns the stored credential hash.
func (u *User) PasswordHash() PasswordHash { return u.passwordHash }

// Status returns the current user status.
func (u *User) Status() UserStatus { return u.status }

// CreatedAt returns when the user was created.
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt returns when the user was last modified.
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// Version returns the aggregate version for optimistic concurrency and event sourcing.
func (u *User) Version() int { return u.version }

// UncommittedEvents returns domain events not yet persisted.
func (u *User) UncommittedEvents() []Event {
	if len(u.events) == 0 {
		return nil
	}
	out := make([]Event, len(u.events))
	copy(out, u.events)
	return out
}

// ClearUncommittedEvents resets the uncommitted event buffer after persistence.
func (u *User) ClearUncommittedEvents() {
	u.events = nil
}

func isValidUserStatus(status UserStatus) bool {
	return status == UserStatusActive || status == UserStatusDisabled
}
