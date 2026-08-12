package domain

// TokenStatus is the lifecycle state of a stored refresh token.
type TokenStatus string

const (
	TokenStatusActive   TokenStatus = "active"
	TokenStatusConsumed TokenStatus = "consumed"
	TokenStatusRevoked  TokenStatus = "revoked"
	TokenStatusExpired  TokenStatus = "expired"
)

// ParseTokenStatus validates a persisted token status.
func ParseTokenStatus(raw string) (TokenStatus, error) {
	switch TokenStatus(raw) {
	case TokenStatusActive, TokenStatusConsumed, TokenStatusRevoked, TokenStatusExpired:
		return TokenStatus(raw), nil
	default:
		return "", ErrInvalidRefreshToken
	}
}
