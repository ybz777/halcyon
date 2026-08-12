package domain

// APIKeyStatus is the lifecycle state of an API key.
type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusDisabled APIKeyStatus = "disabled"
	APIKeyStatusRevoked  APIKeyStatus = "revoked"
	APIKeyStatusExpired  APIKeyStatus = "expired"
)

// ParseAPIKeyStatus validates a persisted API key status.
func ParseAPIKeyStatus(raw string) (APIKeyStatus, error) {
	switch APIKeyStatus(raw) {
	case APIKeyStatusActive, APIKeyStatusDisabled, APIKeyStatusRevoked, APIKeyStatusExpired:
		return APIKeyStatus(raw), nil
	default:
		return "", ErrInvalidAPIKey
	}
}

// ServiceAccountStatus is the lifecycle state of a service account.
type ServiceAccountStatus string

const (
	ServiceAccountStatusActive   ServiceAccountStatus = "active"
	ServiceAccountStatusDisabled ServiceAccountStatus = "disabled"
)

// ParseServiceAccountStatus validates a persisted service account status.
func ParseServiceAccountStatus(raw string) (ServiceAccountStatus, error) {
	switch ServiceAccountStatus(raw) {
	case ServiceAccountStatusActive, ServiceAccountStatusDisabled:
		return ServiceAccountStatus(raw), nil
	default:
		return "", ErrInvalidServiceAccount
	}
}
