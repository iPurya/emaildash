package inbox

import (
	"context"

	"github.com/purya/emaildash/backend/internal/domain"
)

type Store interface {
	ListEmails(ctx context.Context, recipient string, unreadOnly bool, limit int) ([]domain.Email, error)
	GetEmail(ctx context.Context, id int64) (domain.Email, error)
	MarkEmailRead(ctx context.Context, id int64) error
	ListRecipients(ctx context.Context) ([]domain.RecipientSummary, error)
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

func (s Service) ListEmails(ctx context.Context, recipient string, unreadOnly bool, limit int) ([]domain.Email, error) {
	return s.store.ListEmails(ctx, recipient, unreadOnly, limit)
}

func (s Service) GetEmail(ctx context.Context, id int64) (domain.Email, error) {
	return s.store.GetEmail(ctx, id)
}

func (s Service) MarkRead(ctx context.Context, id int64) error {
	return s.store.MarkEmailRead(ctx, id)
}

func (s Service) ListRecipients(ctx context.Context) ([]domain.RecipientSummary, error) {
	return s.store.ListRecipients(ctx)
}
