package ports

import (
	"context"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, string, error)
	Verify(password, hash, params string) error
}

type SecretSealer interface {
	Seal(plaintext string) (string, error)
	Open(ciphertext string) (string, error)
}

type RequestSigner interface {
	Sign(secret []byte, timestamp string, body []byte) string
	Verify(secret []byte, timestamp, signature string, body []byte, maxSkew time.Duration) error
}

type CloudflareClient interface {
	ListZones(ctx context.Context, creds domain.CloudflareCredentials) ([]domain.CloudflareZone, error)
	GetZone(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) (domain.CloudflareZone, error)
	GetEmailRoutingDNS(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) ([]domain.DNSRecord, error)
	EnableEmailRouting(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) error
	EnsureWorkerSubdomain(ctx context.Context, creds domain.CloudflareCredentials, accountID, subdomain string) error
	UploadWorker(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName, scriptContents string) error
	PutWorkerSecret(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName, name, value string) error
	EnableWorkersDev(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName string) error
	UpdateCatchAllToWorker(ctx context.Context, creds domain.CloudflareCredentials, zoneID, scriptName string) error
	GetCatchAllStatus(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) (domain.CloudflareStatus, error)
}
