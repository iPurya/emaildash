package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/ports"
)

const webhookSecretKey = "webhook_signing_secret"

type Store interface {
	GetSecret(ctx context.Context, key string) (string, error)
	InsertEmail(ctx context.Context, email domain.Email) (domain.Email, error)
}

type Service struct {
	store         Store
	sealer        ports.SecretSealer
	signer        ports.RequestSigner
	attachmentDir string
}

func NewService(store Store, sealer ports.SecretSealer, signer ports.RequestSigner, attachmentDir string) Service {
	return Service{store: store, sealer: sealer, signer: signer, attachmentDir: attachmentDir}
}

func (s Service) Ingest(ctx context.Context, timestamp, signature string, rawBody []byte, payload domain.IngestPayload) (domain.Email, error) {
	secret, err := s.secret(ctx)
	if err != nil {
		return domain.Email{}, err
	}
	if err := s.signer.Verify([]byte(secret), timestamp, signature, rawBody, 5*time.Minute); err != nil {
		return domain.Email{}, err
	}
	if payload.Provider == "" {
		payload.Provider = "cloudflare"
	}
	if payload.MessageID == "" {
		payload.MessageID = fallbackMessageID(payload)
	}
	if payload.ReceivedAt.IsZero() {
		payload.ReceivedAt = time.Now().UTC()
	}
	attachments, err := s.persistAttachments(payload.MessageID, payload.Attachments)
	if err != nil {
		return domain.Email{}, err
	}
	email := domain.Email{
		Provider:          payload.Provider,
		ProviderMessageID: payload.MessageID,
		MailFrom:          strings.TrimSpace(payload.MailFrom),
		Recipients:        payload.RcptTo,
		Subject:           payload.Subject,
		TextBody:          payload.Text,
		HTMLBody:          payload.HTML,
		Headers:           payload.Headers,
		RawSize:           payload.RawSize,
		ReceivedAt:        payload.ReceivedAt.UTC(),
		Attachments:       attachments,
	}
	return s.store.InsertEmail(ctx, email)
}

func (s Service) secret(ctx context.Context) (string, error) {
	sealed, err := s.store.GetSecret(ctx, webhookSecretKey)
	if err != nil {
		return "", fmt.Errorf("webhook secret not configured")
	}
	return s.sealer.Open(sealed)
}

func (s Service) persistAttachments(messageID string, attachments []domain.Attachment) ([]domain.Attachment, error) {
	if err := os.MkdirAll(s.attachmentDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir attachments: %w", err)
	}
	stored := make([]domain.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		filename := safeFilename(attachment.Filename)
		path := filepath.Join(s.attachmentDir, messageID+"-"+filename)
		if attachment.Content != "" {
			if err := os.WriteFile(path, []byte(attachment.Content), 0o600); err != nil {
				return nil, fmt.Errorf("write attachment: %w", err)
			}
		}
		attachment.StoragePath = path
		attachment.Content = ""
		stored = append(stored, attachment)
	}
	return stored, nil
}

func fallbackMessageID(payload domain.IngestPayload) string {
	sum := sha256.Sum256([]byte(payload.MailFrom + "|" + strings.Join(payload.RcptTo, ",") + "|" + payload.Subject + "|" + payload.Text + "|" + payload.HTML))
	return hex.EncodeToString(sum[:])
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" {
		return "attachment.bin"
	}
	return strings.ReplaceAll(name, "..", "_")
}
