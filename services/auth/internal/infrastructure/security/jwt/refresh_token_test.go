package jwt_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestService_IssueAndValidateRefreshToken(t *testing.T) {
	t.Parallel()

	service, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_integrationtest0004")
	family, _ := domain.NewFamilyID()

	token, claims, err := service.IssueRefreshToken(context.Background(), family, nil, userID)
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}

	validated, err := service.ValidateRefreshToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}
	if !validated.JTI.Equal(claims.JTI) || !validated.FamilyID.Equal(family) || !validated.UserID.Equal(userID) {
		t.Fatalf("claims mismatch: %+v", validated)
	}
}

func TestService_ValidateRefreshToken_Expired(t *testing.T) {
	t.Parallel()

	service, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, -time.Second)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_integrationtest0005")
	family, _ := domain.NewFamilyID()
	token, _, err := service.IssueRefreshToken(context.Background(), family, nil, userID)
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}

	_, err = service.ValidateRefreshToken(context.Background(), token)
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("error = %v, want ErrTokenExpired", err)
	}
}
