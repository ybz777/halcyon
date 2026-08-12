package events

import "errors"

var (
	// ErrInvalidEnvelope indicates a malformed event envelope.
	ErrInvalidEnvelope = errors.New("invalid event envelope")
	// ErrUnsupportedVersion indicates an unknown event type or version.
	ErrUnsupportedVersion = errors.New("unsupported event version")
)
