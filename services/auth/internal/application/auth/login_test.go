package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestLoginUseCase_InvalidCredentials(t *testing.T) {
	t.Parallel()

	jwtService := mustJWTService(t)
	uc := auth.NewLoginUseCase(
		stubUserRepo{},
		stubRoleRepo{},
		newMemoryRefreshRepo(),
		stubHasher{ok: false},
		jwtService,
		jwtService,
		nil,
		nil,
		nil,
		nil,
	)

	_, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "missing@example.com",
		Password: "password",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginUseCase_Success(t *testing.T) {
	t.Parallel()

	id, _ := domain.ParseUserID("usr_integrationtest0101")
	email, _ := domain.ParseEmail("login@example.com")
	hash, _ := domain.ParsePasswordHash(validBcryptHash)
	user, err := domain.RegisterUser(id, email, hash, time.Now().UTC())
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	jwtService := mustJWTService(t)
	uc := auth.NewLoginUseCase(
		stubUserRepo{user: user},
		stubRoleRepo{},
		newMemoryRefreshRepo(),
		stubHasher{ok: true},
		jwtService,
		jwtService,
		nil,
		nil,
		nil,
		nil,
	)

	result, err := uc.Execute(context.Background(), auth.LoginInput{
		Email:    "login@example.com",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("result = %+v", result)
	}
}

const validBcryptHash = "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.G2oYbOuSO.8K6O"

type stubUserRepo struct {
	user *domain.User
}

func (s stubUserRepo) Create(context.Context, *domain.User) error { return nil }
func (s stubUserRepo) Update(context.Context, *domain.User) error { return nil }
func (s stubUserRepo) Delete(context.Context, domain.UserID) error {
	return nil
}
func (s stubUserRepo) FindByID(context.Context, domain.UserID) (*domain.User, error) {
	return s.user, nil
}
func (s stubUserRepo) FindByEmail(context.Context, domain.Email) (*domain.User, error) {
	if s.user == nil {
		return nil, domain.ErrUserNotFound
	}
	return s.user, nil
}
func (s stubUserRepo) ExistsByEmail(context.Context, domain.Email) (bool, error) {
	return s.user != nil, nil
}

type stubRoleRepo struct{}

func (stubRoleRepo) FindRolesByUserID(context.Context, domain.UserID) ([]domain.Role, error) {
	return nil, nil
}

func (stubRoleRepo) AssignRole(context.Context, domain.UserID, domain.Role) error {
	return nil
}

type stubHasher struct {
	ok bool
}

func (s stubHasher) Hash(context.Context, string) (domain.PasswordHash, error) {
	return domain.PasswordHash{}, nil
}
func (s stubHasher) Verify(context.Context, string, domain.PasswordHash) (bool, error) {
	return s.ok, nil
}
