package jwt_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestService_IssueAndValidateAccessToken(t *testing.T) {
	t.Parallel()

	service, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_integrationtest0001")
	email, _ := domain.ParseEmail("user@example.com")
	role, _ := domain.ParseRole("user")

	token, _, err := service.IssueAccessToken(context.Background(), ports.TokenClaims{
		UserID:      userID,
		Email:       email,
		Roles:       []domain.Role{role},
		Permissions: domain.PermissionsForRoles([]domain.Role{role}),
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	claims, err := service.ValidateAccessToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if !claims.UserID.Equal(userID) {
		t.Fatalf("user id mismatch")
	}
}

func TestService_ValidateAccessToken_InvalidSignature(t *testing.T) {
	t.Parallel()

	issuer, _ := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	validator, _ := jwtinfra.NewService([]byte("different-secret-key-32-bytes!!!!"), "auth", time.Minute, time.Hour)

	userID, _ := domain.ParseUserID("usr_integrationtest0002")
	email, _ := domain.ParseEmail("user2@example.com")
	token, _, err := issuer.IssueAccessToken(context.Background(), ports.TokenClaims{
		UserID: userID,
		Email:  email,
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	_, err = validator.ValidateAccessToken(context.Background(), token)
	if !errors.Is(err, domain.ErrInvalidAccessToken) {
		t.Fatalf("error = %v, want ErrInvalidAccessToken", err)
	}
}

func TestService_ValidateAccessToken_Expired(t *testing.T) {
	t.Parallel()

	service, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", -time.Second, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_integrationtest0003")
	email, _ := domain.ParseEmail("user3@example.com")
	token, _, err := service.IssueAccessToken(context.Background(), ports.TokenClaims{
		UserID: userID,
		Email:  email,
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	_, err = service.ValidateAccessToken(context.Background(), token)
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("error = %v, want ErrTokenExpired", err)
	}
}

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	token, err := jwtinfra.ExtractBearerToken("Bearer abc.def.ghi")
	if err != nil {
		t.Fatalf("ExtractBearerToken() error = %v", err)
	}
	if token.String() != "abc.def.ghi" {
		t.Fatalf("token = %q", token.String())
	}
}
