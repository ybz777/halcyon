package domain

import "errors"

var (
	ErrInvalidNotificationID = errors.New("invalid notification id")
	ErrInvalidEmailDeliveryID = errors.New("invalid email delivery id")
	ErrNotificationNotFound  = errors.New("notification not found")
	ErrForbidden             = errors.New("forbidden")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrInvalidCursor         = errors.New("invalid cursor")
	ErrInvalidPreference     = errors.New("invalid notification preference")
)
