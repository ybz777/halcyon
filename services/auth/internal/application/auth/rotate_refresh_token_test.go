package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestRotateRefreshTokenHandler_Rotation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jwtService := mustJWTService(t)
	user := mustRefreshUser(t)
	repo := newMemoryRefreshRepo()

	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, mustFamilyID(), nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	seedActiveToken(t, repo, refreshToken, claims)

	handler := newRotateHandler(t, jwtService, repo, user)
	result, err := handler.Handle(ctx, auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("result = %+v", result)
	}
	if repo.consumeCount != 1 {
		t.Fatalf("consume=%d", repo.consumeCount)
	}
}

func TestRotateRefreshTokenHandler_ReplayDetection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jwtService := mustJWTService(t)
	user := mustRefreshUser(t)
	repo := newMemoryRefreshRepo()
	family := mustFamilyID()

	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, family, nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	seedConsumedToken(t, repo, refreshToken, claims)

	handler := newRotateHandler(t, jwtService, repo, user)
	_, err = handler.Handle(ctx, auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()})
	if !errors.Is(err, domain.ErrRefreshTokenReused) {
		t.Fatalf("error = %v, want ErrRefreshTokenReused", err)
	}
	if !repo.familyRevoked {
		t.Fatal("expected family revocation")
	}
}

func TestRotateRefreshTokenHandler_ExpiredToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jwtService, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, -time.Second)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	user := mustRefreshUser(t)
	handler := newRotateHandler(t, jwtService, newMemoryRefreshRepo(), user)

	refreshToken, _, err := jwtService.IssueRefreshToken(ctx, mustFamilyID(), nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}

	_, err = handler.Handle(ctx, auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()})
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Fatalf("error = %v, want ErrTokenExpired", err)
	}
}

func TestRotateRefreshTokenHandler_ConcurrentRefresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jwtService := mustJWTService(t)
	user := mustRefreshUser(t)
	repo := newMemoryRefreshRepo()

	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, mustFamilyID(), nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	seedActiveToken(t, repo, refreshToken, claims)

	handler := newRotateHandler(t, jwtService, repo, user)
	cmd := auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := handler.Handle(ctx, cmd)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successes int
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want 1", successes)
	}
}

func TestRevokeTokenFamilyHandler(t *testing.T) {
	t.Parallel()

	repo := newMemoryRefreshRepo()
	handler := auth.NewRevokeTokenFamilyHandler(repo, nil)
	if err := handler.Handle(context.Background(), auth.RevokeTokenFamilyCommand{FamilyID: mustFamilyID()}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !repo.familyRevoked {
		t.Fatal("expected family revoked")
	}
}

func newRotateHandler(t *testing.T, jwtService *jwtinfra.Service, repo *memoryRefreshRepo, user *domain.User) *auth.RotateRefreshTokenHandler {
	t.Helper()
	revoke := auth.NewRevokeTokenFamilyHandler(repo, nil)
	return auth.NewRotateRefreshTokenHandler(
		stubUserRepo{user: user},
		stubRoleRepo{},
		repo,
		jwtService,
		jwtService,
		revoke,
		nil,
	)
}

func mustJWTService(t *testing.T) *jwtinfra.Service {
	t.Helper()
	service, err := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func mustFamilyID() domain.FamilyID {
	family, _ := domain.NewFamilyID()
	return family
}

func mustRefreshUser(t *testing.T) *domain.User {
	t.Helper()
	id, _ := domain.ParseUserID("usr_integrationtest0401")
	email, _ := domain.ParseEmail("refresh@example.com")
	hash, _ := domain.ParsePasswordHash(validBcryptHash)
	user, err := domain.RegisterUser(id, email, hash, time.Now().UTC())
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	return user
}

func seedActiveToken(t *testing.T, repo *memoryRefreshRepo, token domain.RefreshToken, claims ports.RefreshTokenClaims) {
	t.Helper()
	record, err := domain.NewRefreshTokenRecord(
		claims.JTI, claims.FamilyID, nil, claims.UserID,
		domain.HashRefreshToken(token), claims.IssuedAt, claims.ExpiresAt, "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := repo.Save(context.Background(), record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func seedConsumedToken(t *testing.T, repo *memoryRefreshRepo, token domain.RefreshToken, claims ports.RefreshTokenClaims) {
	t.Helper()
	record, err := domain.NewRefreshTokenRecord(
		claims.JTI, claims.FamilyID, nil, claims.UserID,
		domain.HashRefreshToken(token), claims.IssuedAt, claims.ExpiresAt, "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := record.MarkConsumed(time.Now().UTC()); err != nil {
		t.Fatalf("MarkConsumed() error = %v", err)
	}
	if err := repo.Save(context.Background(), record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}
