package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

// RegisterUserCommand carries user registration input.
type RegisterUserCommand struct {
	Email    string
	Password string
}

// RegisterUserHandler registers a new user and emits UserRegistered.
type RegisterUserHandler struct {
	users    ports.UserRepository
	roles    ports.UserRoleRepository
	hasher   ports.PasswordHasher
	tx       ports.TransactionManager
	events   ports.EventPublisher
}

// NewRegisterUserHandler creates a RegisterUserHandler.
func NewRegisterUserHandler(
	users ports.UserRepository,
	roles ports.UserRoleRepository,
	hasher ports.PasswordHasher,
	tx ports.TransactionManager,
	events ports.EventPublisher,
) *RegisterUserHandler {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}

	return &RegisterUserHandler{
		users:  users,
		roles:  roles,
		hasher: hasher,
		tx:     tx,
		events: events,
	}
}

// Handle registers a user atomically with a default user role and outbox event.
func (h *RegisterUserHandler) Handle(ctx context.Context, cmd RegisterUserCommand) (domain.UserID, error) {
	email, err := domain.ParseEmail(cmd.Email)
	if err != nil {
		return domain.UserID{}, err
	}

	exists, err := h.users.ExistsByEmail(ctx, email)
	if err != nil {
		return domain.UserID{}, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return domain.UserID{}, domain.ErrUserAlreadyExists
	}

	hash, err := h.hasher.Hash(ctx, cmd.Password)
	if err != nil {
		return domain.UserID{}, fmt.Errorf("hash password: %w", err)
	}

	id, err := domain.NewUserID()
	if err != nil {
		return domain.UserID{}, err
	}

	now := time.Now().UTC()

	user, err := domain.RegisterUser(id, email, hash, now)
	if err != nil {
		return domain.UserID{}, err
	}

	defaultRole, err := domain.NewRole(domain.RoleUser, "")
	if err != nil {
		return domain.UserID{}, fmt.Errorf("create default user role: %w", err)
	}

	err = h.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := h.users.Create(txCtx, user); err != nil {
			return err
		}

		if h.roles != nil {
			if err := h.roles.AssignRole(txCtx, user.ID(), defaultRole); err != nil {
				return err
			}
		}

		for _, event := range user.UncommittedEvents() {
			if err := infraevents.PublishDomainEvent(txCtx, h.events, event, ""); err != nil {
				return fmt.Errorf("publish event: %w", err)
			}
		}

		user.ClearUncommittedEvents()
		return nil
	})
	if err != nil {
		return domain.UserID{}, err
	}

	return user.ID(), nil
}