package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func toDomainUser(row sqlcgen.User) (*domain.User, error) {
	id, err := domain.ParseUserID(row.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	email, err := domain.ParseEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("parse email: %w", err)
	}

	hash, err := domain.ParsePasswordHash(row.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("parse password hash: %w", err)
	}

	status := domain.UserStatus(row.Status)
	createdAt := timestamptzToTime(row.CreatedAt)
	updatedAt := timestamptzToTime(row.UpdatedAt)

	return domain.ReconstituteUser(id, email, hash, status, createdAt, updatedAt, int(row.Version))
}

func fromDomainUser(user *domain.User) sqlcgen.CreateUserParams {
	return sqlcgen.CreateUserParams{
		ExternalID:   user.ID().String(),
		Email:        user.Email().String(),
		PasswordHash: user.PasswordHash().String(),
		Status:       string(user.Status()),
		Version:      int32(user.Version()),
		CreatedAt:    timeToTimestamptz(user.CreatedAt()),
		UpdatedAt:    timeToTimestamptz(user.UpdatedAt()),
	}
}

func updateParamsFromUser(user *domain.User) sqlcgen.UpdateUserParams {
	return sqlcgen.UpdateUserParams{
		ExternalID:   user.ID().String(),
		Email:        user.Email().String(),
		PasswordHash: user.PasswordHash().String(),
		Status:       string(user.Status()),
		Version:      int32(user.Version()),
		UpdatedAt:    timeToTimestamptz(user.UpdatedAt()),
		OldVersion:   int32(user.Version() - 1),
	}
}

func timeToTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func timestamptzToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time.UTC()
}

func mapSQLError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrUserAlreadyExists
	}

	return err
}
