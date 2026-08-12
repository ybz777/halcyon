package notification

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type Repository interface {
	FindByID(ctx context.Context, orgID, userID string, id domain.NotificationID) (domain.Notification, error)
	ListByUser(ctx context.Context, orgID, userID string, limit int, cursor string) ([]domain.Notification, string, error)
	UnreadCount(ctx context.Context, orgID, userID string) (int64, error)
	MarkRead(ctx context.Context, orgID, userID string, id domain.NotificationID, at time.Time) error
	MarkAllRead(ctx context.Context, orgID, userID string, at time.Time) (int64, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, orgID, userID string, limit int, cursor string) ([]domain.Notification, string, error) {
	return s.repo.ListByUser(ctx, orgID, userID, limit, cursor)
}

func (s *Service) Get(ctx context.Context, orgID, userID string, id domain.NotificationID) (domain.Notification, error) {
	return s.repo.FindByID(ctx, orgID, userID, id)
}

func (s *Service) UnreadCount(ctx context.Context, orgID, userID string) (int64, error) {
	return s.repo.UnreadCount(ctx, orgID, userID)
}

func (s *Service) MarkRead(ctx context.Context, orgID, userID string, id domain.NotificationID) error {
	return s.repo.MarkRead(ctx, orgID, userID, id, time.Now().UTC())
}

func (s *Service) MarkAllRead(ctx context.Context, orgID, userID string) (int64, error) {
	return s.repo.MarkAllRead(ctx, orgID, userID, time.Now().UTC())
}
