package setup

import (
	"context"
	"fmt"

	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/ports"
)

type Store interface {
	SetupStatus(ctx context.Context) (domain.SetupStatus, error)
	InitializeUser(ctx context.Context, hash, params string) error
}

type Service struct {
	store  Store
	hasher ports.PasswordHasher
}

func NewService(store Store, hasher ports.PasswordHasher) Service {
	return Service{store: store, hasher: hasher}
}

func (s Service) Status(ctx context.Context) (domain.SetupStatus, error) {
	return s.store.SetupStatus(ctx)
}

func (s Service) Initialize(ctx context.Context, password string) error {
	hash, params, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.store.InitializeUser(ctx, hash, params)
}
