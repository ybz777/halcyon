package domain

import "time"

// Event is a domain event emitted by an aggregate. Designed for future event
// sourcing: events are immutable facts with aggregate identity and occurrence time.
type Event interface {
	EventType() string
	AggregateID() UserID
	OccurredAt() time.Time
}

// UserRegistered is emitted when a new user account is created.
type UserRegistered struct {
	UserID UserID
	Email  Email
	at     time.Time
}

func (e UserRegistered) EventType() string { return "auth.user.registered" }

func (e UserRegistered) AggregateID() UserID { return e.UserID }

func (e UserRegistered) OccurredAt() time.Time { return e.at }

// NewUserRegistered constructs a UserRegistered event.
func NewUserRegistered(userID UserID, email Email, at time.Time) UserRegistered {
	return UserRegistered{UserID: userID, Email: email, at: at.UTC()}
}

// UserLoggedIn is emitted when a user successfully authenticates.
type UserLoggedIn struct {
	UserID UserID
	Email  Email
	at     time.Time
}

func (e UserLoggedIn) EventType() string { return "auth.user.logged_in" }

func (e UserLoggedIn) AggregateID() UserID { return e.UserID }

func (e UserLoggedIn) OccurredAt() time.Time { return e.at }

// NewUserLoggedIn constructs a UserLoggedIn event.
func NewUserLoggedIn(userID UserID, email Email, at time.Time) UserLoggedIn {
	return UserLoggedIn{UserID: userID, Email: email, at: at.UTC()}
}

// UserLoginFailed is emitted when authentication fails for a known reason category.
type UserLoginFailed struct {
	UserID UserID
	Reason string
	at     time.Time
}

func (e UserLoginFailed) EventType() string { return "auth.user.login_failed" }

func (e UserLoginFailed) AggregateID() UserID { return e.UserID }

func (e UserLoginFailed) OccurredAt() time.Time { return e.at }

// NewUserLoginFailed constructs a UserLoginFailed event.
func NewUserLoginFailed(userID UserID, reason string, at time.Time) UserLoginFailed {
	return UserLoginFailed{UserID: userID, Reason: reason, at: at.UTC()}
}

// UserEmailChanged is emitted when a user updates their email address.
type UserEmailChanged struct {
	UserID   UserID
	OldEmail Email
	NewEmail Email
	at       time.Time
}

func (e UserEmailChanged) EventType() string { return "auth.user.email_changed" }

func (e UserEmailChanged) AggregateID() UserID { return e.UserID }

func (e UserEmailChanged) OccurredAt() time.Time { return e.at }

// NewUserEmailChanged constructs a UserEmailChanged event.
func NewUserEmailChanged(userID UserID, oldEmail, newEmail Email, at time.Time) UserEmailChanged {
	return UserEmailChanged{
		UserID:   userID,
		OldEmail: oldEmail,
		NewEmail: newEmail,
		at:       at.UTC(),
	}
}

// UserDisabled is emitted when a user account is deactivated.
type UserDisabled struct {
	UserID UserID
	at     time.Time
}

func (e UserDisabled) EventType() string { return "auth.user.disabled" }

func (e UserDisabled) AggregateID() UserID { return e.UserID }

func (e UserDisabled) OccurredAt() time.Time { return e.at }

// NewUserDisabled constructs a UserDisabled event.
func NewUserDisabled(userID UserID, at time.Time) UserDisabled {
	return UserDisabled{UserID: userID, at: at.UTC()}
}

// UserEnabled is emitted when a disabled user account is reactivated.
type UserEnabled struct {
	UserID UserID
	at     time.Time
}

func (e UserEnabled) EventType() string { return "auth.user.enabled" }

func (e UserEnabled) AggregateID() UserID { return e.UserID }

func (e UserEnabled) OccurredAt() time.Time { return e.at }

// NewUserEnabled constructs a UserEnabled event.
func NewUserEnabled(userID UserID, at time.Time) UserEnabled {
	return UserEnabled{UserID: userID, at: at.UTC()}
}
