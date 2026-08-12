package jwt



import (

	"context"

	"strings"



	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"

	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"

	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"

)



type Validator struct {

	inner *platformauth.HS256Validator

}



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



func (v *Validator) Validate(_ context.Context, bearerToken string) (appwebhook.Principal, error) {

	claims, err := v.inner.ValidateBearer(bearerToken)

	if err != nil {

		return appwebhook.Principal{}, domain.ErrUnauthorized

	}

	perms := make(map[string]struct{}, len(claims.Permissions))

	for _, perm := range claims.Permissions {

		perms[strings.ToLower(strings.TrimSpace(perm))] = struct{}{}

	}

	return appwebhook.Principal{UserID: claims.UserID, Permissions: perms}, nil

}

