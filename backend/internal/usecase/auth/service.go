package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/ports"
)

type Store interface {
	GetUser(ctx context.Context) (domain.User, error)
	UpdatePassword(ctx context.Context, hash, params string) error
	CreateSession(ctx context.Context, tokenHash, csrfToken string, expiresAt time.Time) (domain.Session, error)
	GetSessionByHash(ctx context.Context, tokenHash string) (domain.Session, error)
	TouchSession(ctx context.Context, id int64) error
	RevokeSession(ctx context.Context, id int64) error
}

type Service struct {
	store      Store
	hasher     ports.PasswordHasher
	sessionTTL time.Duration
}

func NewService(store Store, hasher ports.PasswordHasher, sessionTTL time.Duration) Service {
	return Service{store: store, hasher: hasher, sessionTTL: sessionTTL}
}

func (s Service) Login(ctx context.Context, password string) (string, domain.Session, error) {
	user, err := s.store.GetUser(ctx)
	if err != nil {
		return "", domain.Session{}, err
	}
	if err := s.hasher.Verify(password, user.PasswordHash, user.PasswordParams); err != nil {
		return "", domain.Session{}, fmt.Errorf("invalid credentials")
	}
	token, err := randomToken(32)
	if err != nil {
		return "", domain.Session{}, err
	}
	csrfToken, err := randomToken(24)
	if err != nil {
		return "", domain.Session{}, err
	}
	session, err := s.store.CreateSession(ctx, hashToken(token), csrfToken, time.Now().UTC().Add(s.sessionTTL))
	if err != nil {
		return "", domain.Session{}, err
	}
	return token, session, nil
}

func (s Service) Authenticate(ctx context.Context, token string) (domain.Session, error) {
	session, err := s.store.GetSessionByHash(ctx, hashToken(token))
	if err != nil {
		return domain.Session{}, fmt.Errorf("invalid session")
	}
	if session.RevokedAt != nil {
		return domain.Session{}, fmt.Errorf("session revoked")
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return domain.Session{}, fmt.Errorf("session expired")
	}
	if err := s.store.TouchSession(ctx, session.ID); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s Service) Logout(ctx context.Context, token string) error {
	session, err := s.store.GetSessionByHash(ctx, hashToken(token))
	if err != nil {
		return nil
	}
	return s.store.RevokeSession(ctx, session.ID)
}

func (s Service) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	user, err := s.store.GetUser(ctx)
	if err != nil {
		return err
	}
	if err := s.hasher.Verify(oldPassword, user.PasswordHash, user.PasswordParams); err != nil {
		return fmt.Errorf("invalid current password")
	}
	hash, params, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, hash, params)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
