package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/purya/emaildash/backend/internal/domain"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() Client {
	return Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
}

func (c Client) ListZones(ctx context.Context, creds domain.CloudflareCredentials) ([]domain.CloudflareZone, error) {
	var response struct {
		Result []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Account struct {
				ID string `json:"id"`
			} `json:"account"`
		} `json:"result"`
	}
	if err := c.get(ctx, creds, "/zones?per_page=100", &response); err != nil {
		return nil, err
	}
	zones := make([]domain.CloudflareZone, 0, len(response.Result))
	for _, item := range response.Result {
		zones = append(zones, domain.CloudflareZone{ID: item.ID, Name: item.Name, AccountID: item.Account.ID})
	}
	return zones, nil
}

func (c Client) GetZone(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) (domain.CloudflareZone, error) {
	var response struct {
		Result struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Account struct {
				ID string `json:"id"`
			} `json:"account"`
		} `json:"result"`
	}
	if err := c.get(ctx, creds, "/zones/"+zoneID, &response); err != nil {
		return domain.CloudflareZone{}, err
	}
	return domain.CloudflareZone{ID: response.Result.ID, Name: response.Result.Name, AccountID: response.Result.Account.ID}, nil
}

func (c Client) GetEmailRoutingDNS(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) ([]domain.DNSRecord, error) {
	var response struct {
		Result struct {
			Records []struct {
				Type     string `json:"type"`
				Name     string `json:"name"`
				Content  string `json:"content"`
				Priority int    `json:"priority"`
				TTL      int    `json:"ttl"`
			} `json:"records"`
		} `json:"result"`
	}
	if err := c.get(ctx, creds, "/zones/"+zoneID+"/email/routing/dns", &response); err != nil {
		return nil, err
	}
	out := make([]domain.DNSRecord, 0, len(response.Result.Records))
	for _, record := range response.Result.Records {
		out = append(out, domain.DNSRecord{Type: record.Type, Name: record.Name, Content: record.Content, Priority: record.Priority, TTL: record.TTL})
	}
	return out, nil
}

func (c Client) EnableEmailRouting(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) error {
	return c.post(ctx, creds, "/zones/"+zoneID+"/email/routing/enable", map[string]any{}, nil)
}

func (c Client) EnsureWorkerSubdomain(ctx context.Context, creds domain.CloudflareCredentials, accountID, subdomain string) error {
	err := c.put(ctx, creds, "/accounts/"+accountID+"/workers/subdomain", map[string]any{"subdomain": subdomain}, nil)
	if err != nil && strings.Contains(err.Error(), "already has an associated subdomain") {
		return nil
	}
	return err
}

func (c Client) UploadWorker(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName, scriptContents string) error {
	metadata, err := json.Marshal(map[string]any{"main_module": "index.js"})
	if err != nil {
		return fmt.Errorf("marshal worker metadata: %w", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadataHeader := textproto.MIMEHeader{}
	metadataHeader.Set("Content-Disposition", `form-data; name="metadata"`)
	metadataHeader.Set("Content-Type", "application/json")
	metadataPart, err := writer.CreatePart(metadataHeader)
	if err != nil {
		return fmt.Errorf("create metadata part: %w", err)
	}
	if _, err := metadataPart.Write(metadata); err != nil {
		return fmt.Errorf("write metadata part: %w", err)
	}
	scriptHeader := textproto.MIMEHeader{}
	scriptHeader.Set("Content-Disposition", `form-data; name="index.js"; filename="index.js"`)
	scriptHeader.Set("Content-Type", "application/javascript+module")
	scriptPart, err := writer.CreatePart(scriptHeader)
	if err != nil {
		return fmt.Errorf("create script part: %w", err)
	}
	if _, err := scriptPart.Write([]byte(scriptContents)); err != nil {
		return fmt.Errorf("write script part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}
	return c.doRequest(ctx, http.MethodPut, creds, "/accounts/"+accountID+"/workers/scripts/"+scriptName, &body, writer.FormDataContentType(), nil)
}

func (c Client) PutWorkerSecret(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName, name, value string) error {
	return c.put(ctx, creds, "/accounts/"+accountID+"/workers/scripts/"+scriptName+"/secrets", map[string]string{"type": "secret_text", "name": name, "text": value}, nil)
}

func (c Client) EnableWorkersDev(ctx context.Context, creds domain.CloudflareCredentials, accountID, scriptName string) error {
	return c.post(ctx, creds, "/accounts/"+accountID+"/workers/scripts/"+scriptName+"/subdomain", map[string]any{"enabled": true}, nil)
}

func (c Client) UpdateCatchAllToWorker(ctx context.Context, creds domain.CloudflareCredentials, zoneID, scriptName string) error {
	body := map[string]any{
		"matchers": []map[string]string{{"type": "all"}},
		"actions": []map[string]any{{"type": "worker", "value": []string{scriptName}}},
		"enabled": true,
		"name":    "catch-all",
	}
	return c.put(ctx, creds, "/zones/"+zoneID+"/email/routing/rules/catch_all", body, nil)
}

func (c Client) GetCatchAllStatus(ctx context.Context, creds domain.CloudflareCredentials, zoneID string) (domain.CloudflareStatus, error) {
	var response struct {
		Result []struct {
			Actions []struct {
				Type  string   `json:"type"`
				Value []string `json:"value"`
			} `json:"actions"`
			Enabled bool   `json:"enabled"`
			Name    string `json:"name"`
		} `json:"result"`
	}
	if err := c.get(ctx, creds, "/zones/"+zoneID+"/email/routing/rules", &response); err != nil {
		return domain.CloudflareStatus{}, err
	}
	status := domain.CloudflareStatus{EmailRoutingEnabled: len(response.Result) > 0, EmailRoutingStatus: "configured"}
	for _, rule := range response.Result {
		if strings.EqualFold(rule.Name, "catch-all") || (rule.Name == "" && len(rule.Actions) > 0) {
			status.CatchAllEnabled = rule.Enabled
			for _, action := range rule.Actions {
				if action.Type == "worker" && len(action.Value) > 0 {
					status.CatchAllDestination = action.Value[0]
				}
			}
		}
	}
	if !status.CatchAllEnabled {
		status.EmailRoutingStatus = "pending"
	}
	return status, nil
}

func (c Client) get(ctx context.Context, creds domain.CloudflareCredentials, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, creds, path, nil, out)
}

func (c Client) post(ctx context.Context, creds domain.CloudflareCredentials, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, creds, path, body, out)
}

func (c Client) put(ctx context.Context, creds domain.CloudflareCredentials, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPut, creds, path, body, out)
}

func (c Client) doJSON(ctx context.Context, method string, creds domain.CloudflareCredentials, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal cloudflare body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	return c.doRequest(ctx, method, creds, path, reader, "application/json", out)
}

func (c Client) doRequest(ctx context.Context, method string, creds domain.CloudflareCredentials, path string, body io.Reader, contentType string, out any) error {
	endpoint, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("X-Auth-Email", creds.Email)
	req.Header.Set("X-Auth-Key", creds.APIKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	var wrapper struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(responseBody, &wrapper); err == nil {
		if !wrapper.Success && len(wrapper.Errors) > 0 {
			return fmt.Errorf("cloudflare api: %s", wrapper.Errors[0].Message)
		}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("cloudflare api: %s", strings.TrimSpace(string(responseBody)))
	}
	if out != nil {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode cloudflare response: %w", err)
		}
	}
	return nil
}
