package dto

import appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"

// RegisterRequest is the user registration endpoint request body.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the login endpoint request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the refresh endpoint request body.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// TokenResponse is the token pair returned by login and refresh.
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

// ToTokenResponse maps an application token response to the API DTO.
func ToTokenResponse(result appauth.TokenResponse) TokenResponse {
	return TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	}
}