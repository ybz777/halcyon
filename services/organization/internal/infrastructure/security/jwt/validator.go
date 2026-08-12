package jwt



import (

	"context"



	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"

)



// Validator validates auth service JWT access tokens.

type Validator struct {

	inner *platformauth.HS256Validator

}



// NewValidator creates a JWT validator.

func NewValidator(secret []byte, issuer, audience string) (*Validator, error) {

	inner, err := platformauth.NewHS256Validator(platformauth.HS256Config{

		Secret:   secret,

		Issuer:   issuer,

		Audience: audience,

	})

	if err != nil {

		return nil, err

	}

	return &Validator{inner: inner}, nil

}



// Validate parses and validates a bearer token.

func (v *Validator) Validate(_ context.Context, bearerToken string) (ports.AuthenticatedUser, error) {

	claims, err := v.inner.ValidateBearer(bearerToken)

	if err != nil {

		return ports.AuthenticatedUser{}, domain.ErrUnauthorized

	}



	userID, err := domain.ParseUserID(claims.UserID)

	if err != nil {

		return ports.AuthenticatedUser{}, domain.ErrUnauthorized

	}

	email, err := domain.ParseEmail(claims.Email)

	if err != nil {

		return ports.AuthenticatedUser{}, domain.ErrUnauthorized

	}

	return ports.AuthenticatedUser{UserID: userID, Email: email}, nil

}



var _ ports.TokenValidator = (*Validator)(nil)

