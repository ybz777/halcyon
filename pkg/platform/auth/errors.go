package auth

import "errors"

// FailureCategory classifies authentication failures for metrics and logs.
type FailureCategory string

const (
	FailureMissingCredential FailureCategory = "missing_credential"
	FailureMalformedToken    FailureCategory = "malformed_token"
	FailureInvalidToken      FailureCategory = "invalid_token"
	FailureExpiredToken      FailureCategory = "expired_token"
	FailureNotBefore         FailureCategory = "not_before"
	FailureWrongIssuer       FailureCategory = "wrong_issuer"
	FailureWrongAudience     FailureCategory = "wrong_audience"
	FailureWrongAlgorithm    FailureCategory = "wrong_algorithm"
	FailureMissingSubject    FailureCategory = "missing_subject"
	FailureUntrustedGateway  FailureCategory = "untrusted_gateway"
	FailureRevokedCredential FailureCategory = "revoked_credential"
)

var (
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidCredential = errors.New("invalid credential")
)

// AuthError wraps an authentication failure with a stable category.
type AuthError struct {
	Category FailureCategory
	Err      error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Category)
}

func (e *AuthError) Unwrap() error { return e.Err }

// CategoryOf returns the failure category when err is an AuthError.
func CategoryOf(err error) FailureCategory {
	var authErr *AuthError
	if errors.As(err, &authErr) && authErr.Category != "" {
		return authErr.Category
	}
	return FailureInvalidToken
}
