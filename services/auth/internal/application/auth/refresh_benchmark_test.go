package auth_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

func TestRefreshRotationBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load benchmark in short mode")
	}

	const iterations = 10_000
	ctx := context.Background()
	jwtService := mustJWTService(t)
	user := mustRefreshUser(t)
	repo := newMemoryRefreshRepo()
	handler := newRotateHandler(t, jwtService, repo, user)

	refreshToken, claims, err := jwtService.IssueRefreshToken(ctx, mustFamily(t), nil, user.ID())
	if err != nil {
		t.Fatalf("IssueRefreshToken() error = %v", err)
	}
	seedActiveToken(t, repo, refreshToken, claims)

	latencies := make([]time.Duration, 0, iterations)
	var allocs uint64

	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err := handler.Handle(ctx, auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()})
		latencies = append(latencies, time.Since(start))
		if err != nil {
			if i == 0 {
				t.Fatalf("first rotation error = %v", err)
			}
			break
		}
		refreshToken, _ = domain.ParseRefreshToken(result.RefreshToken)
		claims, err = jwtService.ValidateRefreshToken(ctx, refreshToken)
		if err != nil {
			t.Fatalf("ValidateRefreshToken() error = %v", err)
		}
		record, err := domain.NewRefreshTokenRecord(
			claims.JTI, claims.FamilyID, nil, claims.UserID,
			domain.HashRefreshToken(refreshToken), claims.IssuedAt, claims.ExpiresAt, "", "",
		)
		if err != nil {
			t.Fatalf("NewRefreshTokenRecord() error = %v", err)
		}
		if err := repo.Save(ctx, record); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		allocs++
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50 := latencies[len(latencies)*50/100]
	p95 := latencies[len(latencies)*95/100]
	t.Logf("iterations=%d p50=%s p95=%s successful_rotations=%d", len(latencies), p50, p95, allocs)
}

func BenchmarkRotateRefreshToken(b *testing.B) {
	ctx := context.Background()
	jwtService, _ := jwtinfra.NewService([]byte("super-secret-key-with-32-bytes!!"), "auth", time.Minute, time.Hour)
	user, _ := domain.ParseUserID("usr_integrationtest0401")
	email, _ := domain.ParseEmail("refresh@example.com")
	hash, _ := domain.ParsePasswordHash(validBcryptHash)
	domainUser, _ := domain.RegisterUser(user, email, hash, time.Now().UTC())
	repo := newMemoryRefreshRepo()
	revoke := auth.NewRevokeTokenFamilyHandler(repo, nil)
	handler := auth.NewRotateRefreshTokenHandler(
		stubUserRepo{user: domainUser},
		stubRoleRepo{},
		repo,
		jwtService,
		jwtService,
		revoke,
		nil,
	)

	refreshToken, claims, _ := jwtService.IssueRefreshToken(ctx, mustFamilyID(), nil, domainUser.ID())
	record, _ := domain.NewRefreshTokenRecord(
		claims.JTI, claims.FamilyID, nil, claims.UserID,
		domain.HashRefreshToken(refreshToken), claims.IssuedAt, claims.ExpiresAt, "", "",
	)
	_ = repo.Save(ctx, record)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := handler.Handle(ctx, auth.RotateRefreshTokenCommand{RefreshToken: refreshToken.String()})
		if err != nil {
			b.Fatalf("Handle() error = %v", err)
		}
		next, _ := domain.ParseRefreshToken(result.RefreshToken)
		nextClaims, _ := jwtService.ValidateRefreshToken(ctx, next)
		nextRecord, _ := domain.NewRefreshTokenRecord(
			nextClaims.JTI, nextClaims.FamilyID, nil, nextClaims.UserID,
			domain.HashRefreshToken(next), nextClaims.IssuedAt, nextClaims.ExpiresAt, "", "",
		)
		_ = repo.Save(ctx, nextRecord)
		refreshToken = next
	}
}
