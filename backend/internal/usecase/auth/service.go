package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/ports"
)

const apiAccessKeySecret = "api_access_key"

type Store interface {
	GetUser(ctx context.Context) (domain.User, error)
	UpdatePassword(ctx context.Context, hash, params string) error
	CreateSession(ctx context.Context, tokenHash, csrfToken string, expiresAt time.Time) (domain.Session, error)
	GetSessionByHash(ctx context.Context, tokenHash string) (domain.Session, error)
	TouchSession(ctx context.Context, id int64) error
	RevokeSession(ctx context.Context, id int64) error
	PutSecret(ctx context.Context, key, ciphertext string) error
	GetSecret(ctx context.Context, key string) (string, error)
}

type Service struct {
	store      Store
	hasher     ports.PasswordHasher
	sealer     ports.SecretSealer
	sessionTTL time.Duration
}

func NewService(store Store, hasher ports.PasswordHasher, sealer ports.SecretSealer, sessionTTL time.Duration) Service {
	return Service{store: store, hasher: hasher, sealer: sealer, sessionTTL: sessionTTL}
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
	return session, nil
}

func (s Service) AuthenticateAPIKey(ctx context.Context, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("missing api key")
	}
	sealed, err := s.store.GetSecret(ctx, apiAccessKeySecret)
	if err != nil {
		return fmt.Errorf("api key not configured")
	}
	stored, err := s.sealer.Open(sealed)
	if err != nil {
		return fmt.Errorf("open api key: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(apiKey)) != 1 {
		return fmt.Errorf("invalid api key")
	}
	return nil
}

func (s Service) Logout(ctx context.Context, token string) error {
	session, err := s.store.GetSessionByHash(ctx, hashToken(token))
	if err != nil {
		return nil
	}
	return s.store.RevokeSession(ctx, session.ID)
}

func (s Service) ChangePassword(ctx context.Context, oldPassword, newPassword string) (string, error) {
	user, err := s.store.GetUser(ctx)
	if err != nil {
		return "", err
	}
	if err := s.hasher.Verify(oldPassword, user.PasswordHash, user.PasswordParams); err != nil {
		return "", fmt.Errorf("invalid current password")
	}
	hash, params, err := s.hasher.Hash(newPassword)
	if err != nil {
		return "", err
	}
	if err := s.store.UpdatePassword(ctx, hash, params); err != nil {
		return "", err
	}
	return s.RotateAPIKey(ctx)
}

func (s Service) EnsureAPIKey(ctx context.Context) (string, error) {
	sealed, err := s.store.GetSecret(ctx, apiAccessKeySecret)
	if err == nil {
		return s.sealer.Open(sealed)
	}
	return s.RotateAPIKey(ctx)
}

func (s Service) RotateAPIKey(ctx context.Context) (string, error) {
	apiKey, err := randomToken(24)
	if err != nil {
		return "", err
	}
	sealed, err := s.sealer.Seal(apiKey)
	if err != nil {
		return "", err
	}
	if err := s.store.PutSecret(ctx, apiAccessKeySecret, sealed); err != nil {
		return "", err
	}
	return apiKey, nil
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
