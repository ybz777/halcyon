package preference

import (
	"context"
	"fmt"

	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type Repository interface {
	ListByUser(ctx context.Context, userID string, limit int, cursor string) ([]domain.NotificationPreference, string, error)
	Upsert(ctx context.Context, pref domain.NotificationPreference) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) ([]domain.NotificationPreference, string, error) {
	return s.repo.ListByUser(ctx, userID, limit, cursor)
}

func (s *Service) Update(ctx context.Context, userID string, notificationType domain.NotificationType, channel domain.Channel, enabled bool) (domain.NotificationPreference, error) {
	if domain.IsMandatorySecurityType(notificationType) && channel == domain.ChannelEmail && !enabled {
		return domain.NotificationPreference{}, fmt.Errorf("%w: security notifications cannot disable email", domain.ErrInvalidPreference)
	}
	pref := domain.NotificationPreference{
		UserID: userID, NotificationType: notificationType, Channel: channel, Enabled: enabled,
	}
	if err := s.repo.Upsert(ctx, pref); err != nil {
		return domain.NotificationPreference{}, err
	}
	return pref, nil
}
