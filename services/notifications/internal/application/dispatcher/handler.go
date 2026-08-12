package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/notifications/internal/application/mapper"
	"github.com/halcyon-io/halcyon/services/notifications/internal/application/resolver"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobEnqueuer interface {
	Enqueue(ctx context.Context, input platformjobs.CreateInput) (platformjobs.Job, bool, error)
}

type Handler struct {
	notifications *postgres.NotificationRepository
	preferences   *postgres.PreferenceRepository
	pool          *pgxpool.Pool
	group         string
	maxPerEvent   int
	jobs          JobEnqueuer
}

func NewHandler(notifications *postgres.NotificationRepository, preferences *postgres.PreferenceRepository, pool *pgxpool.Pool, group string, maxPerEvent int, jobs JobEnqueuer) *Handler {
	if maxPerEvent <= 0 {
		maxPerEvent = domain.MaxNotificationsPerEvent
	}
	return &Handler{notifications: notifications, preferences: preferences, pool: pool, group: group, maxPerEvent: maxPerEvent, jobs: jobs}
}

func (h *Handler) Handle(ctx context.Context, envelope platformevents.Envelope) error {
	specs, ok := mapper.MapEnvelope(envelope)
	if !ok || envelope.OrganizationID == "" {
		return nil
	}
	recipients := resolver.Resolve(envelope)
	if len(recipients) == 0 {
		return nil
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var claimed bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, organization_id, status)
		VALUES ($1, $2, $3, 'processing')
		ON CONFLICT (event_id, consumer_group) DO NOTHING
		RETURNING true
	`, envelope.EventID, h.group, envelope.OrganizationID).Scan(&claimed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}

	now := time.Now().UTC()
	created := 0
	for _, spec := range specs {
		for _, recipient := range recipients {
			if created >= h.maxPerEvent {
				break
			}
			if recipient.UserID == "" {
				continue
			}
			id, err := domain.NewNotificationID()
			if err != nil {
				return err
			}
			notification := domain.Notification{
				ID: id, OrganizationID: recipient.OrganizationID, UserID: recipient.UserID,
				EventID: envelope.EventID, EventType: envelope.EventType,
				NotificationType: spec.NotificationType, Title: spec.Title, Body: spec.Body,
				ResourceType: spec.ResourceType, ResourceID: spec.ResourceID,
				Metadata: spec.Metadata, CreatedAt: now,
			}
			prefs, err := h.preferences.PreferencesForUser(ctx, recipient.UserID, spec.NotificationType)
			if err != nil {
				return err
			}
			if !domain.IsChannelEnabled(spec.NotificationType, domain.ChannelInApp, prefs) {
				continue
			}
			if err := insertNotification(ctx, tx, notification); err != nil {
				return err
			}

			if domain.IsChannelEnabled(spec.NotificationType, domain.ChannelEmail, prefs) && recipient.Email != "" {
				deliveryID, err := domain.NewEmailDeliveryID()
				if err != nil {
					return err
				}
				if err := insertEmailDelivery(ctx, tx, domain.EmailDelivery{
					ID: deliveryID, NotificationID: id, UserID: recipient.UserID,
					Email: recipient.Email, Template: email.TemplateForType(spec.NotificationType),
					Status: domain.EmailPending, AvailableAt: now, CreatedAt: now,
				}); err != nil {
					return err
				}
				if h.jobs != nil {
					payload, err := platformjobs.MarshalEmailDelivery(deliveryID.String())
					if err != nil {
						return err
					}
					if _, _, err := h.jobs.Enqueue(ctx, platformjobs.CreateInput{
						OrganizationID: envelope.OrganizationID,
						Type:           platformjobs.TypeEmailDelivery,
						Payload:        payload,
						IdempotencyKey: deliveryID.String(),
						Priority:       platformjobs.PriorityNormal,
						MaxAttempts:    6,
						AvailableAt:    now,
					}); err != nil {
						return fmt.Errorf("enqueue email job: %w", err)
					}
				}
			}
			created++
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE processed_events SET status='completed', processed_at=now()
		WHERE event_id=$1 AND consumer_group=$2
	`, envelope.EventID, h.group)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return tx.Commit(ctx)
}

func insertNotification(ctx context.Context, tx pgx.Tx, n domain.Notification) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications (
			id, organization_id, user_id, event_id, event_type, notification_type,
			title, body, resource_type, resource_id, metadata, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'{}',$11)
		ON CONFLICT (event_id, user_id, notification_type) DO NOTHING
	`, n.ID.String(), n.OrganizationID, n.UserID, n.EventID, n.EventType, string(n.NotificationType),
		n.Title, n.Body, n.ResourceType, n.ResourceID, n.CreatedAt.UTC())
	if err != nil {
		return err
	}
	_ = tag
	return nil
}

func insertEmailDelivery(ctx context.Context, tx pgx.Tx, d domain.EmailDelivery) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO email_deliveries (
			id, notification_id, user_id, email, template, status, attempt_count, available_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,0,$7,$7)
	`, d.ID.String(), d.NotificationID.String(), d.UserID, d.Email, d.Template, string(d.Status), d.AvailableAt.UTC())
	return err
}
