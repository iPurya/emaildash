package cloudflare

import (
	"context"
	"fmt"
	"os"

	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/ports"
)

const (
	secretCloudflareEmail      = "cloudflare_email"
	secretCloudflareAPIKey     = "cloudflare_api_key"
	secretWebhookSigningSecret = "webhook_signing_secret"
)

type Store interface {
	PutSecret(ctx context.Context, key, ciphertext string) error
	GetSecret(ctx context.Context, key string) (string, error)
	ReplaceZones(ctx context.Context, zones []domain.CloudflareZone) error
	ListZones(ctx context.Context) ([]domain.CloudflareZone, error)
	SelectZone(ctx context.Context, zoneID, status string) error
	GetSelectedZone(ctx context.Context) (domain.CloudflareZone, error)
	InsertAuditLog(ctx context.Context, eventType string, details map[string]any) error
}

type Service struct {
	store            Store
	client           ports.CloudflareClient
	sealer           ports.SecretSealer
	workerScriptName string
	workerSubdomain  string
	workerBundlePath string
	publicBaseURL    string
}

func NewService(store Store, client ports.CloudflareClient, sealer ports.SecretSealer, workerScriptName, workerSubdomain, workerBundlePath, publicBaseURL string) Service {
	return Service{
		store:            store,
		client:           client,
		sealer:           sealer,
		workerScriptName: workerScriptName,
		workerSubdomain:  workerSubdomain,
		workerBundlePath: workerBundlePath,
		publicBaseURL:    publicBaseURL,
	}
}

func (s Service) SaveCredentials(ctx context.Context, creds domain.CloudflareCredentials) ([]domain.CloudflareZone, error) {
	if creds.Email == "" || creds.APIKey == "" {
		return nil, fmt.Errorf("cloudflare email and api key required")
	}
	encryptedEmail, err := s.sealer.Seal(creds.Email)
	if err != nil {
		return nil, err
	}
	encryptedKey, err := s.sealer.Seal(creds.APIKey)
	if err != nil {
		return nil, err
	}
	if err := s.store.PutSecret(ctx, secretCloudflareEmail, encryptedEmail); err != nil {
		return nil, err
	}
	if err := s.store.PutSecret(ctx, secretCloudflareAPIKey, encryptedKey); err != nil {
		return nil, err
	}
	zones, err := s.client.ListZones(ctx, creds)
	if err != nil {
		return nil, err
	}
	if err := s.store.ReplaceZones(ctx, zones); err != nil {
		return nil, err
	}
	_ = s.store.InsertAuditLog(ctx, "cloudflare.credentials.saved", map[string]any{"zoneCount": len(zones)})
	return zones, nil
}

func (s Service) ListCachedZones(ctx context.Context) ([]domain.CloudflareZone, error) {
	return s.store.ListZones(ctx)
}

func (s Service) ProvisionZone(ctx context.Context, zoneID string) (domain.CloudflareStatus, error) {
	creds, err := s.credentials(ctx)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	zone, err := s.client.GetZone(ctx, creds, zoneID)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	if zone.AccountID == "" {
		return domain.CloudflareStatus{}, fmt.Errorf("zone account id missing")
	}
	if err := s.client.EnableEmailRouting(ctx, creds, zone.ID); err != nil {
		return domain.CloudflareStatus{}, err
	}
	if err := s.client.EnsureWorkerSubdomain(ctx, creds, zone.AccountID, s.workerSubdomain); err != nil {
		return domain.CloudflareStatus{}, err
	}
	bundle, err := os.ReadFile(s.workerBundlePath)
	if err != nil {
		return domain.CloudflareStatus{}, fmt.Errorf("read worker bundle: %w", err)
	}
	if err := s.client.UploadWorker(ctx, creds, zone.AccountID, s.workerScriptName, string(bundle)); err != nil {
		return domain.CloudflareStatus{}, err
	}
	webhookSecret, err := s.ensureWebhookSecret(ctx)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	if err := s.client.PutWorkerSecret(ctx, creds, zone.AccountID, s.workerScriptName, "EMAILDASH_WEBHOOK_URL", s.publicBaseURL+"/api/ingest/cloudflare/email"); err != nil {
		return domain.CloudflareStatus{}, err
	}
	if err := s.client.PutWorkerSecret(ctx, creds, zone.AccountID, s.workerScriptName, "EMAILDASH_WEBHOOK_SECRET", webhookSecret); err != nil {
		return domain.CloudflareStatus{}, err
	}
	if err := s.client.EnableWorkersDev(ctx, creds, zone.AccountID, s.workerScriptName); err != nil {
		return domain.CloudflareStatus{}, err
	}
	if err := s.client.UpdateCatchAllToWorker(ctx, creds, zone.ID, s.workerScriptName); err != nil {
		return domain.CloudflareStatus{}, err
	}
	status, err := s.client.GetCatchAllStatus(ctx, creds, zone.ID)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	status.ZoneID = zone.ID
	status.ZoneName = zone.Name
	status.AccountID = zone.AccountID
	status.WorkerScriptName = s.workerScriptName
	if err := s.store.SelectZone(ctx, zone.ID, status.EmailRoutingStatus); err != nil {
		return domain.CloudflareStatus{}, err
	}
	_ = s.store.InsertAuditLog(ctx, "cloudflare.zone.provisioned", map[string]any{"zoneId": zone.ID, "zoneName": zone.Name})
	return status, nil
}

func (s Service) Status(ctx context.Context) (domain.CloudflareStatus, error) {
	creds, err := s.credentials(ctx)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	zone, err := s.store.GetSelectedZone(ctx)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	status, err := s.client.GetCatchAllStatus(ctx, creds, zone.ID)
	if err != nil {
		return domain.CloudflareStatus{}, err
	}
	status.ZoneID = zone.ID
	status.ZoneName = zone.Name
	status.AccountID = zone.AccountID
	status.WorkerScriptName = s.workerScriptName
	return status, nil
}

func (s Service) WebhookSecret(ctx context.Context) (string, error) {
	return s.ensureWebhookSecret(ctx)
}

func (s Service) credentials(ctx context.Context) (domain.CloudflareCredentials, error) {
	encryptedEmail, err := s.store.GetSecret(ctx, secretCloudflareEmail)
	if err != nil {
		return domain.CloudflareCredentials{}, fmt.Errorf("cloudflare email not configured")
	}
	encryptedKey, err := s.store.GetSecret(ctx, secretCloudflareAPIKey)
	if err != nil {
		return domain.CloudflareCredentials{}, fmt.Errorf("cloudflare api key not configured")
	}
	email, err := s.sealer.Open(encryptedEmail)
	if err != nil {
		return domain.CloudflareCredentials{}, err
	}
	apiKey, err := s.sealer.Open(encryptedKey)
	if err != nil {
		return domain.CloudflareCredentials{}, err
	}
	return domain.CloudflareCredentials{Email: email, APIKey: apiKey}, nil
}

func (s Service) ensureWebhookSecret(ctx context.Context) (string, error) {
	existing, err := s.store.GetSecret(ctx, secretWebhookSigningSecret)
	if err == nil {
		return s.sealer.Open(existing)
	}
	secret, err := randomHex(32)
	if err != nil {
		return "", err
	}
	sealed, err := s.sealer.Seal(secret)
	if err != nil {
		return "", err
	}
	if err := s.store.PutSecret(ctx, secretWebhookSigningSecret, sealed); err != nil {
		return "", err
	}
	return secret, nil
}
