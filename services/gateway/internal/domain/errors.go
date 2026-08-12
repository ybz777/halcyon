package domain

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrRateLimited        = errors.New("rate limited")
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
	ErrRequestTooLarge    = errors.New("request body too large")
)
