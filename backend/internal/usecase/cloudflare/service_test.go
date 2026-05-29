package cloudflare

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
)

func TestReceivingDomainsUsesLiveStatusWhenLocalStatusIsBlank(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	store.zones = []domain.CloudflareZone{
		{ID: "zone-1", Name: "ipurya.ir", AccountID: "account-1"},
		{ID: "zone-2", Name: "nigma.ir", AccountID: "account-1"},
	}
	client := newFakeCloudflareClient()
	client.statuses["zone-1"] = readyStatus("emaildash-ingest")
	client.statuses["zone-2"] = readyStatus("emaildash-ingest")
	service := newTestService(t, store, client)

	domains, err := service.ReceivingDomains(ctx)
	if err != nil {
		t.Fatalf("ReceivingDomains returned error: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(domains))
	}
	for _, item := range domains {
		if !item.Ready {
			t.Fatalf("expected %s to be ready, got reason %q", item.Domain, item.Reason)
		}
	}
	if client.statusCalls != 2 {
		t.Fatalf("expected 2 status calls, got %d", client.statusCalls)
	}

	if _, err := service.ReceivingDomains(ctx); err != nil {
		t.Fatalf("cached ReceivingDomains returned error: %v", err)
	}
	if client.statusCalls != 2 {
		t.Fatalf("expected cached call not to hit Cloudflare, got %d calls", client.statusCalls)
	}
}

func TestEnableReceivingUpdatesOnlyTargetZoneStatus(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	store.zones = []domain.CloudflareZone{
		{ID: "zone-1", Name: "ipurya.ir", AccountID: "account-1"},
		{ID: "zone-2", Name: "nigma.ir", AccountID: "account-1", Selected: true, Status: "configured"},
	}
	client := newFakeCloudflareClient()
	service := newTestService(t, store, client)

	status, err := service.EnableReceiving(ctx, "zone-1")
	if err != nil {
		t.Fatalf("EnableReceiving returned error: %v", err)
	}
	if !status.CatchAllEnabled || status.CatchAllDestination != "emaildash-ingest" {
		t.Fatalf("expected catch-all to point at worker, got %+v", status)
	}
	if store.zones[0].Status != "configured" {
		t.Fatalf("expected target status to be configured, got %q", store.zones[0].Status)
	}
	if !store.zones[1].Selected || store.zones[1].Status != "configured" {
		t.Fatalf("expected non-target selected/status fields to be preserved, got %+v", store.zones[1])
	}
	if client.workerUpdates != 1 {
		t.Fatalf("expected one worker catch-all update, got %d", client.workerUpdates)
	}
}

func TestDisableReceivingClearsCachedReadyDomain(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	store.zones = []domain.CloudflareZone{{ID: "zone-1", Name: "ipurya.ir", AccountID: "account-1"}}
	client := newFakeCloudflareClient()
	client.statuses["zone-1"] = readyStatus("emaildash-ingest")
	service := newTestService(t, store, client)

	domains, err := service.ReceivingDomains(ctx)
	if err != nil {
		t.Fatalf("ReceivingDomains returned error: %v", err)
	}
	if len(domains) != 1 || !domains[0].Ready {
		t.Fatalf("expected cached domain to start ready, got %+v", domains)
	}

	status, err := service.DisableReceiving(ctx, "zone-1")
	if err != nil {
		t.Fatalf("DisableReceiving returned error: %v", err)
	}
	if status.CatchAllEnabled {
		t.Fatalf("expected catch-all disabled, got %+v", status)
	}
	if client.disableCalls != 1 {
		t.Fatalf("expected one disable call, got %d", client.disableCalls)
	}

	domains, err = service.ReceivingDomains(ctx)
	if err != nil {
		t.Fatalf("ReceivingDomains after disable returned error: %v", err)
	}
	if len(domains) != 1 || domains[0].Ready || domains[0].Reason != "catch_all_not_enabled" {
		t.Fatalf("expected disabled domain to be not ready, got %+v", domains)
	}
}

func TestReloadZonesPreservesExistingStateAndAddsNewZones(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	store.zones = []domain.CloudflareZone{
		{ID: "zone-1", Name: "purya.dev", AccountID: "account-1", Selected: true, Status: "configured"},
		{ID: "zone-2", Name: "old.example", AccountID: "account-1", Status: "pending"},
	}
	client := newFakeCloudflareClient()
	client.zones = []domain.CloudflareZone{
		{ID: "zone-1", Name: "purya.dev", AccountID: "account-1"},
		{ID: "zone-3", Name: "new.example", AccountID: "account-1"},
	}
	client.statuses["zone-1"] = readyStatus("emaildash-ingest")
	service := newTestService(t, store, client)

	if _, err := service.ReceivingDomains(ctx); err != nil {
		t.Fatalf("ReceivingDomains returned error: %v", err)
	}
	if client.statusCalls != 2 {
		t.Fatalf("expected first receiving domain lookup to check 2 cached zones, got %d calls", client.statusCalls)
	}

	zones, err := service.ReloadZones(ctx)
	if err != nil {
		t.Fatalf("ReloadZones returned error: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 reloaded zones, got %d", len(zones))
	}
	if zones[0].ID != "zone-1" || !zones[0].Selected || zones[0].Status != "configured" {
		t.Fatalf("expected existing zone state to be preserved, got %+v", zones[0])
	}
	if zones[1].ID != "zone-3" || zones[1].Status != "" || zones[1].Selected {
		t.Fatalf("expected new zone to be added with empty state, got %+v", zones[1])
	}

	if _, err := service.ReceivingDomains(ctx); err != nil {
		t.Fatalf("ReceivingDomains after reload returned error: %v", err)
	}
	if client.statusCalls != 4 {
		t.Fatalf("expected receiving domain cache to be cleared after reload, got %d status calls", client.statusCalls)
	}
}

func newTestService(t *testing.T, store *fakeStore, client *fakeCloudflareClient) Service {
	t.Helper()
	bundle, err := os.CreateTemp(t.TempDir(), "worker-*.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.WriteString("export default {};"); err != nil {
		t.Fatal(err)
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	return NewService(store, client, plainSealer{}, "emaildash-ingest", "emaildash-receiver", bundle.Name(), "https://emaildash.example.test")
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		secrets: map[string]string{
			secretCloudflareEmail:  "admin@example.com",
			secretCloudflareAPIKey: "key",
		},
	}
}

type fakeStore struct {
	secrets map[string]string
	zones   []domain.CloudflareZone
}

func (s *fakeStore) PutSecret(_ context.Context, key, ciphertext string) error {
	s.secrets[key] = ciphertext
	return nil
}

func (s *fakeStore) GetSecret(_ context.Context, key string) (string, error) {
	value, ok := s.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret not found")
	}
	return value, nil
}

func (s *fakeStore) ReplaceZones(_ context.Context, zones []domain.CloudflareZone) error {
	s.zones = append([]domain.CloudflareZone(nil), zones...)
	return nil
}

func (s *fakeStore) ListZones(_ context.Context) ([]domain.CloudflareZone, error) {
	return append([]domain.CloudflareZone(nil), s.zones...), nil
}

func (s *fakeStore) UpdateZoneStatus(_ context.Context, zoneID, status string) error {
	for index := range s.zones {
		if s.zones[index].ID == zoneID {
			s.zones[index].Status = status
			s.zones[index].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return fmt.Errorf("zone not found")
}

func (s *fakeStore) GetSelectedZone(_ context.Context) (domain.CloudflareZone, error) {
	for _, zone := range s.zones {
		if zone.Selected {
			return zone, nil
		}
	}
	return domain.CloudflareZone{}, fmt.Errorf("selected zone not found")
}

func (s *fakeStore) InsertAuditLog(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

type plainSealer struct{}

func (plainSealer) Seal(value string) (string, error) {
	return value, nil
}

func (plainSealer) Open(value string) (string, error) {
	return value, nil
}

func newFakeCloudflareClient() *fakeCloudflareClient {
	return &fakeCloudflareClient{statuses: map[string]domain.CloudflareStatus{}}
}

type fakeCloudflareClient struct {
	statuses      map[string]domain.CloudflareStatus
	zones         []domain.CloudflareZone
	statusCalls   int
	workerUpdates int
	disableCalls  int
}

func (c *fakeCloudflareClient) ListZones(_ context.Context, _ domain.CloudflareCredentials) ([]domain.CloudflareZone, error) {
	return append([]domain.CloudflareZone(nil), c.zones...), nil
}

func (c *fakeCloudflareClient) GetZone(_ context.Context, _ domain.CloudflareCredentials, zoneID string) (domain.CloudflareZone, error) {
	return domain.CloudflareZone{ID: zoneID, Name: zoneID + ".example", AccountID: "account-1"}, nil
}

func (c *fakeCloudflareClient) GetEmailRoutingDNS(_ context.Context, _ domain.CloudflareCredentials, _ string) ([]domain.DNSRecord, error) {
	return nil, nil
}

func (c *fakeCloudflareClient) EnableEmailRouting(_ context.Context, _ domain.CloudflareCredentials, _ string) error {
	return nil
}

func (c *fakeCloudflareClient) EnsureWorkerSubdomain(_ context.Context, _ domain.CloudflareCredentials, _, _ string) error {
	return nil
}

func (c *fakeCloudflareClient) UploadWorker(_ context.Context, _ domain.CloudflareCredentials, _, _, _ string) error {
	return nil
}

func (c *fakeCloudflareClient) PutWorkerSecret(_ context.Context, _ domain.CloudflareCredentials, _, _, _, _ string) error {
	return nil
}

func (c *fakeCloudflareClient) EnableWorkersDev(_ context.Context, _ domain.CloudflareCredentials, _, _ string) error {
	return nil
}

func (c *fakeCloudflareClient) UpdateCatchAllToWorker(_ context.Context, _ domain.CloudflareCredentials, zoneID, scriptName string) error {
	c.workerUpdates++
	c.statuses[zoneID] = readyStatus(scriptName)
	return nil
}

func (c *fakeCloudflareClient) DisableCatchAll(_ context.Context, _ domain.CloudflareCredentials, zoneID string) error {
	c.disableCalls++
	c.statuses[zoneID] = domain.CloudflareStatus{
		EmailRoutingEnabled: true,
		EmailRoutingStatus:  "pending",
		CatchAllEnabled:     false,
	}
	return nil
}

func (c *fakeCloudflareClient) GetCatchAllStatus(_ context.Context, _ domain.CloudflareCredentials, zoneID string) (domain.CloudflareStatus, error) {
	c.statusCalls++
	return c.statuses[zoneID], nil
}

func readyStatus(scriptName string) domain.CloudflareStatus {
	return domain.CloudflareStatus{
		EmailRoutingEnabled: true,
		EmailRoutingStatus:  "configured",
		CatchAllEnabled:     true,
		CatchAllDestination: scriptName,
	}
}
