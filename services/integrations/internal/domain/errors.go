package domain

import "errors"

var (
	ErrInvalidEndpointID  = errors.New("invalid webhook endpoint id")
	ErrInvalidDeliveryID  = errors.New("invalid webhook delivery id")
	ErrEndpointNotFound   = errors.New("webhook endpoint not found")
	ErrDeliveryNotFound   = errors.New("webhook delivery not found")
	ErrForbidden          = errors.New("forbidden")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidURL         = errors.New("invalid webhook url")
	ErrInvalidSubscription = errors.New("invalid event subscription")
	ErrEndpointDisabled   = errors.New("webhook endpoint disabled")
	ErrInvalidCursor      = errors.New("invalid cursor")
)

const (
	PermissionIntegrationsRead  = "integrations.read"
	PermissionIntegrationsWrite = "integrations.write"
	PermissionAPIKeysRead       = "api_keys.read"
	PermissionAPIKeysWrite      = "api_keys.write"
	PermissionAPIKeysRevoke     = "api_keys.revoke"
)

const (
	MaxEndpointsPerOrg    = 25
	MaxSubscriptions      = 50
	MaxDeliveriesPerEvent = 100
)
