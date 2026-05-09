package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	ui "github.com/purya/emaildash/backend/internal/adapters/http/views"
)

func requestBaseURL(c *gin.Context) string {
	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
	}

	return scheme + "://" + host
}

func apiDocsData(baseURL string) ui.APIDocsData {
	return ui.APIDocsData{
		BaseURL:          baseURL,
		OpenAPIURL:       baseURL + "/api/docs/openapi.json",
		AgentMarkdownURL: baseURL + "/api/docs/agent.md",
		SampleAuth:       "Authorization: Bearer YOUR_API_KEY",
		AgentPrompt:      agentPrompt(baseURL),
		AgentRules: []string{
			"Use the API key from the dashboard Password & API page. For agents, prefer the Authorization header over the api_key query parameter.",
			"All request and response bodies are JSON except 204 No Content responses and the markdown/OpenAPI docs.",
			"Date-time values are RFC3339 strings. Treat missing nullable fields as null or absent.",
			"Use limit on list endpoints. Default email list limit is 50.",
			"Do not call /api/ingest/cloudflare/email directly unless you are the configured Cloudflare Worker and can sign the webhook payload.",
			"Never invent endpoints. If an operation is not listed here or in the OpenAPI JSON, it is not part of the supported API.",
		},
		Workflow: []ui.DocsStep{
			{Title: "Check readiness", Body: "Call GET /api/setup/status. If initialized is false, setup must be completed before protected endpoints can be used."},
			{Title: "Authenticate", Body: "Send Authorization: Bearer YOUR_API_KEY on protected endpoints. X-API-Key and ?api_key= are accepted for compatibility."},
			{Title: "Find receiving domains", Body: "Call GET /api/domains. Use readyDomains for the simple list of domains configured and ready to receive email."},
			{Title: "Discover inboxes", Body: "Call GET /api/recipients to find recipient addresses, unread counts, and latest message hints."},
			{Title: "List messages", Body: "Call GET /api/emails with recipient, to_mail, from_mail, unread, and limit filters as needed."},
			{Title: "Read one message", Body: "Call GET /api/emails/{id}. Prefer textBody for plain text processing; use htmlBody only when HTML context matters."},
			{Title: "Mark processed", Body: "After your agent has handled a message, call PATCH /api/emails/{id}/read to mark it read."},
		},
		Endpoints: docsEndpoints(baseURL),
		Schemas:   docsSchemas(),
	}
}

func docsEndpoints(baseURL string) []ui.DocsEndpoint {
	authHeader := `-H "Authorization: Bearer YOUR_API_KEY"`
	jsonHeader := `-H "Content-Type: application/json"`

	return []ui.DocsEndpoint{
		{
			Method:      "GET",
			Path:        "/api/setup/status",
			Description: "Returns whether EmailDash has completed first-time setup.",
			Auth:        "public",
			Request:     fmt.Sprintf("curl %s/api/setup/status", baseURL),
			Response:    `{"initialized":true}`,
			AgentUse:    "Call this first. If initialized is false, protected endpoints will not be useful yet.",
		},
		{
			Method:      "POST",
			Path:        "/api/setup/initialize",
			Description: "Initializes a fresh instance with the first dashboard password. Only works before setup is complete.",
			Auth:        "public before setup",
			Request:     fmt.Sprintf(`curl -X POST %s -d '{"password":"STRONG_PASSWORD"}' %s/api/setup/initialize`, jsonHeader, baseURL),
			Response:    "HTTP 201 Created",
			AgentUse:    "Use only during automated first install. After setup, use the dashboard to copy the API key for future calls.",
		},
		{
			Method:      "POST",
			Path:        "/api/auth/login",
			Description: "Creates a browser session cookie from the dashboard password.",
			Auth:        "public",
			Request:     fmt.Sprintf(`curl -c cookies.txt %s -d '{"password":"YOUR_PASSWORD"}' %s/api/auth/login`, jsonHeader, baseURL),
			Response:    `{"csrfToken":"...","expiresAt":"2026-05-08T22:00:00Z"}`,
			AgentUse:    "Agents should normally skip password login and use an API key instead.",
		},
		{
			Method:      "GET",
			Path:        "/api/auth/me",
			Description: "Verifies the current auth context.",
			Auth:        "cookie or API key",
			Request:     fmt.Sprintf(`curl %s %s/api/auth/me`, authHeader, baseURL),
			Response:    `{"authenticated":true,"auth":"apiKey"}`,
			AgentUse:    "Use this to validate that the API key is accepted before running a longer workflow.",
		},
		{
			Method:      "GET",
			Path:        "/api/domains",
			Description: "Checks every cached Cloudflare zone and returns the domains ready to receive EmailDash mail.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s %s/api/domains`, authHeader, baseURL),
			Response:    `{"readyDomains":["example.com"],"domains":[{"domain":"example.com","zoneId":"ZONE_ID","ready":true,"reason":"ready","emailRoutingEnabled":true,"catchAllEnabled":true,"catchAllDestination":"emaildash-ingest","workerScriptName":"emaildash-ingest"}]}`,
			AgentUse:    "Use this endpoint when the agent needs to know which domains are configured and ready to receive email. The fastest answer is readyDomains.",
		},
		{
			Method:      "POST",
			Path:        "/api/auth/logout",
			Description: "Clears the current browser session cookie.",
			Auth:        "cookie",
			Request:     fmt.Sprintf(`curl -X POST -b cookies.txt %s/api/auth/logout`, baseURL),
			Response:    "HTTP 204 No Content",
			AgentUse:    "Not needed for API-key agents.",
		},
		{
			Method:      "GET",
			Path:        "/api/recipients",
			Description: "Lists recipient inbox summaries with counts and latest-message hints.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s %s/api/recipients`, authHeader, baseURL),
			Response:    `{"recipients":[{"address":"test@example.com","count":12,"unreadCount":3,"latestEmailId":42,"latestSubject":"Hello","latestReceived":"2026-05-08T21:00:00Z"}]}`,
			AgentUse:    "Use this before listing messages so the agent can choose the right recipient address.",
		},
		{
			Method:      "GET",
			Path:        "/api/emails",
			Description: "Lists messages. Query filters: recipient, to_mail, from_mail, unread=true, received_after, limit.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s "%s/api/emails?recipient=test@example.com&received_after=2026-05-09T10:00:00Z&unread=true&limit=25"`, authHeader, baseURL),
			Response:    `{"emails":[{"id":42,"mailFrom":"alice@example.com","recipients":["test@example.com"],"subject":"Hello","readAt":null,"receivedAt":"2026-05-09T10:00:03Z","attachments":[]}]}`,
			AgentUse:    "For polling a temporary address, use received_after with the time the address was issued plus a bounded limit. Then fetch selected IDs with GET /api/emails/{id}.",
		},
		{
			Method:      "GET",
			Path:        "/api/emails/{id}",
			Description: "Returns one full email by numeric ID, including textBody, htmlBody, headers, and attachment metadata.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s %s/api/emails/42`, authHeader, baseURL),
			Response:    `{"id":42,"mailFrom":"alice@example.com","recipients":["test@example.com"],"subject":"Hello","textBody":"Plain text","htmlBody":"<p>Plain text</p>","headers":{"Message-ID":["..."]},"attachments":[]}`,
			AgentUse:    "Use textBody for extraction and classification when present. Use headers for provenance and threading.",
		},
		{
			Method:      "PATCH",
			Path:        "/api/emails/{id}/read",
			Description: "Marks one email as read.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl -X PATCH %s %s/api/emails/42/read`, authHeader, baseURL),
			Response:    "HTTP 204 No Content",
			AgentUse:    "Call only after the agent has successfully processed the message.",
		},
		{
			Method:      "GET",
			Path:        "/api/cloudflare/status",
			Description: "Returns current Cloudflare email routing and worker status for the selected zone.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s %s/api/cloudflare/status`, authHeader, baseURL),
			Response:    `{"zoneName":"example.com","emailRoutingEnabled":true,"emailRoutingStatus":"ready","workerScriptName":"emaildash-ingest","catchAllEnabled":true}`,
			AgentUse:    "Use this before changing routing so the agent understands current infrastructure state.",
		},
		{
			Method:      "GET",
			Path:        "/api/cloudflare/zones",
			Description: "Returns cached Cloudflare zones after credentials have been saved.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl %s %s/api/cloudflare/zones`, authHeader, baseURL),
			Response:    `{"zones":[{"id":"ZONE_ID","name":"example.com","accountId":"ACCOUNT_ID","selected":true,"status":"active"}]}`,
			AgentUse:    "Use this to find the zoneId required by the provision endpoint.",
		},
		{
			Method:      "POST",
			Path:        "/api/cloudflare/credentials",
			Description: "Saves Cloudflare account email and Global API key, then refreshes zones.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl -X POST %s %s -d '{"email":"admin@example.com","apiKey":"CLOUDFLARE_GLOBAL_API_KEY"}' %s/api/cloudflare/credentials`, authHeader, jsonHeader, baseURL),
			Response:    `{"zones":[{"id":"ZONE_ID","name":"example.com","accountId":"ACCOUNT_ID","selected":false,"status":"active"}]}`,
			AgentUse:    "This stores a secret. Agents should ask for explicit operator intent before calling it.",
		},
		{
			Method:      "POST",
			Path:        "/api/cloudflare/zones/{zoneId}/provision",
			Description: "Provisions Email Routing, catch-all routing, worker script, and worker secrets for one Cloudflare zone.",
			Auth:        "API key or cookie",
			Request:     fmt.Sprintf(`curl -X POST %s %s/api/cloudflare/zones/ZONE_ID/provision`, authHeader, baseURL),
			Response:    `{"zoneId":"ZONE_ID","zoneName":"example.com","workerScriptName":"emaildash-ingest","catchAllEnabled":true}`,
			AgentUse:    "This changes DNS/email routing. Agents should confirm the selected zone before calling it.",
		},
		{
			Method:      "POST",
			Path:        "/api/settings/password",
			Description: "Changes the dashboard password and rotates the EmailDash API key.",
			Auth:        "API key or cookie plus old password",
			Request:     fmt.Sprintf(`curl -X POST %s %s -d '{"oldPassword":"OLD_PASSWORD","newPassword":"NEW_PASSWORD"}' %s/api/settings/password`, authHeader, jsonHeader, baseURL),
			Response:    `{"apiKey":"NEW_EMAILDASH_API_KEY"}`,
			AgentUse:    "After success, discard the old API key and store the returned new key.",
		},
		{
			Method:      "POST",
			Path:        "/api/ingest/cloudflare/email",
			Description: "Signed webhook receiver used by the deployed Cloudflare Worker.",
			Auth:        "signed webhook",
			Request:     `Headers: X-Emaildash-Timestamp and X-Emaildash-Signature. Signature is HMAC-SHA256 over "timestamp.rawJsonBody" with value "v1=<hex>".`,
			Response:    "HTTP 201 Created with Email JSON",
			AgentUse:    "Do not use this for normal inbox automation. It exists for mail delivery only.",
		},
	}
}

func docsSchemas() []ui.DocsSchema {
	return []ui.DocsSchema{
		{Name: "ReceivingDomain", Description: "Configured domain readiness returned by /api/domains.", Fields: []ui.DocsField{
			{Name: "domain", Type: "string", Description: "Domain name that can receive email when ready is true."},
			{Name: "zoneId", Type: "string", Description: "Cloudflare zone ID."},
			{Name: "ready", Type: "boolean", Description: "True when Email Routing and catch-all worker routing are both active."},
			{Name: "reason", Type: "string", Description: "Machine-readable readiness reason, such as ready, catch_all_not_enabled, or status_check_failed."},
			{Name: "statusError", Type: "string", Description: "Optional Cloudflare status-check error for this domain."},
			{Name: "emailRoutingEnabled", Type: "boolean", Description: "Whether Cloudflare Email Routing appears enabled."},
			{Name: "emailRoutingStatus", Type: "string", Description: "Cloudflare routing status as known by EmailDash."},
			{Name: "catchAllEnabled", Type: "boolean", Description: "Whether catch-all routing is enabled."},
			{Name: "catchAllDestination", Type: "string", Description: "Worker script that receives catch-all mail."},
			{Name: "workerScriptName", Type: "string", Description: "EmailDash worker script expected to receive email."},
		}},
		{Name: "Email", Description: "Primary message object returned by /api/emails and /api/emails/{id}.", Fields: []ui.DocsField{
			{Name: "id", Type: "integer", Description: "Stable numeric EmailDash ID."},
			{Name: "provider", Type: "string", Description: "Provider that delivered the email, usually cloudflare."},
			{Name: "providerMessageId", Type: "string", Description: "Message ID from the provider or generated fallback."},
			{Name: "mailFrom", Type: "string", Description: "Envelope sender address."},
			{Name: "recipients", Type: "string[]", Description: "Recipient addresses."},
			{Name: "subject", Type: "string", Description: "Email subject, possibly empty."},
			{Name: "textBody", Type: "string", Description: "Plain text body. Prefer this for agent parsing when available."},
			{Name: "htmlBody", Type: "string", Description: "Sanitized for UI display, but API returns stored HTML string."},
			{Name: "headers", Type: "object<string,string[]>", Description: "Email headers grouped by header name."},
			{Name: "rawSize", Type: "integer", Description: "Original raw message size in bytes."},
			{Name: "readAt", Type: "string|null", Description: "RFC3339 time when marked read. Missing/null means unread."},
			{Name: "receivedAt", Type: "string", Description: "RFC3339 receive time."},
			{Name: "attachments", Type: "Attachment[]", Description: "Attachment metadata. Binary content is not returned by read endpoints."},
		}},
		{Name: "Attachment", Description: "Attachment metadata attached to an email.", Fields: []ui.DocsField{
			{Name: "id", Type: "integer", Description: "Stable attachment ID."},
			{Name: "filename", Type: "string", Description: "Original or fallback filename."},
			{Name: "contentType", Type: "string", Description: "MIME type."},
			{Name: "size", Type: "integer", Description: "Size in bytes."},
			{Name: "sha256", Type: "string", Description: "Hex SHA-256 digest of attachment bytes."},
		}},
		{Name: "RecipientSummary", Description: "Mailbox summary returned by /api/recipients.", Fields: []ui.DocsField{
			{Name: "address", Type: "string", Description: "Recipient email address."},
			{Name: "count", Type: "integer", Description: "Total messages for this recipient."},
			{Name: "unreadCount", Type: "integer", Description: "Unread messages for this recipient."},
			{Name: "latestEmailId", Type: "integer|null", Description: "Newest email ID when present."},
			{Name: "latestSubject", Type: "string|null", Description: "Newest email subject when present."},
			{Name: "latestReceived", Type: "string|null", Description: "RFC3339 receive time for newest message."},
		}},
	}
}

func agentPrompt(baseURL string) string {
	return fmt.Sprintf(`EmailDash API instructions for an AI agent

Base URL: %s
Preferred auth for protected endpoints: Authorization: Bearer YOUR_API_KEY
Fallback auth: X-API-Key: YOUR_API_KEY or ?api_key=YOUR_API_KEY
Content type for JSON requests: application/json

Main workflow:
1. GET /api/setup/status
2. GET /api/auth/me with the API key to validate credentials
3. GET /api/domains to list configured domains ready to receive email
4. GET /api/recipients to discover inbox addresses
5. GET /api/emails?recipient=<address>&received_after=<issued_at>&limit=25 to find actionable messages
6. GET /api/emails/{id} for full body and headers
7. PATCH /api/emails/{id}/read only after successful processing

Rules:
- Use only endpoints listed on this page or in /api/docs/openapi.json.
- Treat all timestamps as RFC3339.
- Prefer textBody over htmlBody for semantic extraction.
- Keep API keys out of logs and user-visible messages.
- Do not call Cloudflare provisioning or credential endpoints without explicit operator intent.
- Do not call /api/ingest/cloudflare/email unless acting as the configured signed Cloudflare Worker.`, baseURL)
}

func (h PagesHandler) OpenAPISpec(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, openAPISpec(requestBaseURL(c)))
}

func (h PagesHandler) AgentMarkdown(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(agentMarkdown(requestBaseURL(c))))
}

func agentMarkdown(baseURL string) string {
	var builder strings.Builder
	data := apiDocsData(baseURL)
	builder.WriteString("# EmailDash API Agent Guide\n\n")
	builder.WriteString(data.AgentPrompt)
	builder.WriteString("\n\n## Endpoints\n\n")
	for _, endpoint := range data.Endpoints {
		builder.WriteString(fmt.Sprintf("### %s %s\n\n", endpoint.Method, endpoint.Path))
		builder.WriteString(endpoint.Description + "\n\n")
		builder.WriteString("Auth: " + endpoint.Auth + "\n\n")
		builder.WriteString("Request:\n```bash\n" + endpoint.Request + "\n```\n\n")
		builder.WriteString("Response:\n```text\n" + endpoint.Response + "\n```\n\n")
		builder.WriteString("Agent use: " + endpoint.AgentUse + "\n\n")
	}
	return builder.String()
}

func openAPISpec(baseURL string) gin.H {
	protectedSecurity := []gin.H{{"bearerAuth": []string{}}, {"apiKeyHeader": []string{}}, {"apiKeyQuery": []string{}}}
	jsonObject := gin.H{"application/json": gin.H{"schema": gin.H{"type": "object"}}}
	errorResponse := gin.H{"description": "Error response", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Error"}}}}

	return gin.H{
		"openapi": "3.1.0",
		"info": gin.H{
			"title":       "EmailDash API",
			"version":     "1.0.0",
			"description": "API for EmailDash inbox automation, Cloudflare routing setup, and dashboard auth. AI agents should use bearer API-key auth for protected endpoints.",
		},
		"servers": []gin.H{{"url": baseURL}},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth":   gin.H{"type": "http", "scheme": "bearer", "description": "Preferred for agents. Value is the EmailDash API key."},
				"apiKeyHeader": gin.H{"type": "apiKey", "in": "header", "name": "X-API-Key"},
				"apiKeyQuery":  gin.H{"type": "apiKey", "in": "query", "name": "api_key"},
			},
			"schemas": openAPISchemas(),
		},
		"paths": gin.H{
			"/api/setup/status": gin.H{"get": gin.H{
				"summary":     "Check setup status",
				"description": "Returns whether the instance has completed first-time setup.",
				"security":    []gin.H{},
				"responses": gin.H{
					"200": gin.H{"description": "Setup status", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/SetupStatus"}}}},
					"500": errorResponse,
				},
			}},
			"/api/setup/initialize": gin.H{"post": gin.H{
				"summary":     "Initialize fresh instance",
				"description": "Sets the first dashboard password. Only valid before setup has completed.",
				"security":    []gin.H{},
				"requestBody": gin.H{"required": true, "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/InitializeRequest"}}}},
				"responses":   gin.H{"201": gin.H{"description": "Created"}, "400": errorResponse},
			}},
			"/api/auth/login": gin.H{"post": gin.H{
				"summary":     "Create browser session",
				"description": "Logs in with the dashboard password and sets an HTTP-only cookie. Agents should usually use API-key auth instead.",
				"security":    []gin.H{},
				"requestBody": gin.H{"required": true, "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/LoginRequest"}}}},
				"responses":   gin.H{"200": gin.H{"description": "Session metadata", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/LoginResponse"}}}}, "401": errorResponse},
			}},
			"/api/auth/logout": gin.H{"post": gin.H{
				"summary":     "Clear browser session",
				"description": "Clears the dashboard session cookie.",
				"responses":   gin.H{"204": gin.H{"description": "No Content"}},
			}},
			"/api/auth/me": gin.H{"get": gin.H{
				"summary":     "Verify authentication",
				"description": "Returns whether the current cookie or API key is accepted.",
				"security":    protectedSecurity,
				"responses":   gin.H{"200": gin.H{"description": "Authenticated", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/AuthMeResponse"}}}}, "401": errorResponse},
			}},
			"/api/domains": gin.H{"get": gin.H{
				"summary":     "List ready receiving domains",
				"description": "Returns domains configured for EmailDash email receiving. Use readyDomains for a simple string list of domains where ready is true.",
				"security":    protectedSecurity,
				"responses": gin.H{
					"200": gin.H{"description": "Receiving domains", "content": gin.H{"application/json": gin.H{"schema": gin.H{"type": "object", "properties": gin.H{
						"readyDomains": gin.H{"type": "array", "items": gin.H{"type": "string"}},
						"domains":      gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/ReceivingDomain"}},
					}}}}},
					"401": errorResponse,
				},
			}},
			"/api/recipients": gin.H{"get": gin.H{
				"summary":     "List recipient summaries",
				"description": "Returns grouped recipient addresses with total and unread counts.",
				"security":    protectedSecurity,
				"responses":   gin.H{"200": gin.H{"description": "Recipients", "content": gin.H{"application/json": gin.H{"schema": gin.H{"type": "object", "properties": gin.H{"recipients": gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/RecipientSummary"}}}}}}}, "401": errorResponse},
			}},
			"/api/emails": gin.H{"get": gin.H{
				"summary":     "List emails",
				"description": "Lists messages with optional recipient, sender, unread, and limit filters.",
				"security":    protectedSecurity,
				"parameters": []gin.H{
					queryParam("recipient", "string", "Recipient address filter. Also used as to_mail fallback."),
					queryParam("to_mail", "string", "Recipient address filter."),
					queryParam("from_mail", "string", "Sender address filter."),
					queryParam("unread", "boolean", "When true, return unread messages only."),
					queryParam("received_after", "string", "RFC3339 timestamp. Returns messages received at or after this time."),
					queryParam("limit", "integer", "Maximum rows to return. Default is 50."),
				},
				"responses": gin.H{"200": gin.H{"description": "Emails", "content": gin.H{"application/json": gin.H{"schema": gin.H{"type": "object", "properties": gin.H{"emails": gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/Email"}}}}}}}, "401": errorResponse},
			}},
			"/api/emails/{id}": gin.H{"get": gin.H{
				"summary":     "Get one email",
				"description": "Returns one full email by numeric ID.",
				"security":    protectedSecurity,
				"parameters":  []gin.H{pathParam("id", "integer", "Email ID")},
				"responses":   gin.H{"200": gin.H{"description": "Email", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Email"}}}}, "400": errorResponse, "401": errorResponse, "404": errorResponse},
			}},
			"/api/emails/{id}/read": gin.H{"patch": gin.H{
				"summary":     "Mark email read",
				"description": "Marks one email as read.",
				"security":    protectedSecurity,
				"parameters":  []gin.H{pathParam("id", "integer", "Email ID")},
				"responses":   gin.H{"204": gin.H{"description": "No Content"}, "400": errorResponse, "401": errorResponse},
			}},
			"/api/cloudflare/status": gin.H{"get": gin.H{
				"summary":     "Get Cloudflare status",
				"description": "Returns email routing and worker status for the selected Cloudflare zone.",
				"security":    protectedSecurity,
				"responses":   gin.H{"200": gin.H{"description": "Cloudflare status", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/CloudflareStatus"}}}}, "400": errorResponse, "401": errorResponse},
			}},
			"/api/cloudflare/zones": gin.H{"get": gin.H{
				"summary":     "List Cloudflare zones",
				"description": "Returns cached Cloudflare zones after credentials have been saved.",
				"security":    protectedSecurity,
				"responses":   gin.H{"200": gin.H{"description": "Zones", "content": gin.H{"application/json": gin.H{"schema": gin.H{"type": "object", "properties": gin.H{"zones": gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/CloudflareZone"}}}}}}}, "401": errorResponse},
			}},
			"/api/cloudflare/credentials": gin.H{"post": gin.H{
				"summary":     "Save Cloudflare credentials",
				"description": "Stores Cloudflare account email and Global API key, then refreshes zones.",
				"security":    protectedSecurity,
				"requestBody": gin.H{"required": true, "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/CloudflareCredentialsRequest"}}}},
				"responses":   gin.H{"200": gin.H{"description": "Zones", "content": gin.H{"application/json": gin.H{"schema": gin.H{"type": "object", "properties": gin.H{"zones": gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/CloudflareZone"}}}}}}}, "400": errorResponse, "401": errorResponse},
			}},
			"/api/cloudflare/zones/{zoneId}/provision": gin.H{"post": gin.H{
				"summary":     "Provision Cloudflare zone",
				"description": "Provisions Email Routing, catch-all route, worker script, and worker secrets for one zone.",
				"security":    protectedSecurity,
				"parameters":  []gin.H{pathParam("zoneId", "string", "Cloudflare zone ID")},
				"responses":   gin.H{"200": gin.H{"description": "Cloudflare status", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/CloudflareStatus"}}}}, "400": errorResponse, "401": errorResponse},
			}},
			"/api/settings/password": gin.H{"post": gin.H{
				"summary":     "Change password and rotate API key",
				"description": "Changes dashboard password and returns the new EmailDash API key.",
				"security":    protectedSecurity,
				"requestBody": gin.H{"required": true, "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/ChangePasswordRequest"}}}},
				"responses":   gin.H{"200": gin.H{"description": "New API key", "content": jsonObject}, "400": errorResponse, "401": errorResponse},
			}},
			"/api/ingest/cloudflare/email": gin.H{"post": gin.H{
				"summary":     "Signed Cloudflare Worker webhook",
				"description": "Receives signed parsed email payloads from the configured Cloudflare Worker. Not intended for normal agent use.",
				"parameters": []gin.H{
					headerParam("X-Emaildash-Timestamp", "string", "Unix timestamp in seconds."),
					headerParam("X-Emaildash-Signature", "string", `HMAC-SHA256 signature in format v1=<hex> over "timestamp.rawJsonBody".`),
				},
				"requestBody": gin.H{"required": true, "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/IngestPayload"}}}},
				"responses":   gin.H{"201": gin.H{"description": "Created email", "content": gin.H{"application/json": gin.H{"schema": gin.H{"$ref": "#/components/schemas/Email"}}}}, "400": errorResponse, "401": errorResponse},
			}},
		},
	}
}

func queryParam(name, schemaType, description string) gin.H {
	return gin.H{"name": name, "in": "query", "required": false, "description": description, "schema": gin.H{"type": schemaType}}
}

func pathParam(name, schemaType, description string) gin.H {
	return gin.H{"name": name, "in": "path", "required": true, "description": description, "schema": gin.H{"type": schemaType}}
}

func headerParam(name, schemaType, description string) gin.H {
	return gin.H{"name": name, "in": "header", "required": true, "description": description, "schema": gin.H{"type": schemaType}}
}

func openAPISchemas() gin.H {
	stringArray := gin.H{"type": "array", "items": gin.H{"type": "string"}}
	dateTime := gin.H{"type": "string", "format": "date-time"}
	nullableDateTime := gin.H{"anyOf": []gin.H{dateTime, gin.H{"type": "null"}}}

	return gin.H{
		"Error":             gin.H{"type": "object", "properties": gin.H{"error": gin.H{"type": "string"}}},
		"SetupStatus":       gin.H{"type": "object", "properties": gin.H{"initialized": gin.H{"type": "boolean"}}, "required": []string{"initialized"}},
		"InitializeRequest": gin.H{"type": "object", "properties": gin.H{"password": gin.H{"type": "string"}}, "required": []string{"password"}},
		"LoginRequest":      gin.H{"type": "object", "properties": gin.H{"password": gin.H{"type": "string"}}, "required": []string{"password"}},
		"LoginResponse": gin.H{"type": "object", "properties": gin.H{
			"csrfToken": gin.H{"type": "string"},
			"expiresAt": dateTime,
		}},
		"AuthMeResponse": gin.H{"type": "object", "properties": gin.H{
			"authenticated": gin.H{"type": "boolean"},
			"auth":          gin.H{"type": "string", "enum": []string{"session", "apiKey"}},
			"expiresAt":     dateTime,
			"csrfToken":     gin.H{"type": "string"},
		}},
		"Attachment": gin.H{"type": "object", "properties": gin.H{
			"id":          gin.H{"type": "integer", "format": "int64"},
			"filename":    gin.H{"type": "string"},
			"contentType": gin.H{"type": "string"},
			"size":        gin.H{"type": "integer", "format": "int64"},
			"sha256":      gin.H{"type": "string"},
			"storagePath": gin.H{"type": "string", "description": "Internal path may be omitted from API responses."},
		}},
		"ReceivingDomain": gin.H{"type": "object", "properties": gin.H{
			"domain":              gin.H{"type": "string"},
			"zoneId":              gin.H{"type": "string"},
			"ready":               gin.H{"type": "boolean"},
			"reason":              gin.H{"type": "string"},
			"statusError":         gin.H{"type": "string"},
			"emailRoutingEnabled": gin.H{"type": "boolean"},
			"emailRoutingStatus":  gin.H{"type": "string"},
			"catchAllEnabled":     gin.H{"type": "boolean"},
			"catchAllDestination": gin.H{"type": "string"},
			"workerScriptName":    gin.H{"type": "string"},
		}},
		"Email": gin.H{"type": "object", "properties": gin.H{
			"id":                gin.H{"type": "integer", "format": "int64"},
			"provider":          gin.H{"type": "string"},
			"providerMessageId": gin.H{"type": "string"},
			"mailFrom":          gin.H{"type": "string"},
			"recipients":        stringArray,
			"subject":           gin.H{"type": "string"},
			"textBody":          gin.H{"type": "string"},
			"htmlBody":          gin.H{"type": "string"},
			"headers":           gin.H{"type": "object", "additionalProperties": stringArray},
			"rawSize":           gin.H{"type": "integer", "format": "int64"},
			"readAt":            nullableDateTime,
			"receivedAt":        dateTime,
			"createdAt":         dateTime,
			"attachments":       gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/Attachment"}},
		}},
		"RecipientSummary": gin.H{"type": "object", "properties": gin.H{
			"address":        gin.H{"type": "string"},
			"count":          gin.H{"type": "integer", "format": "int64"},
			"unreadCount":    gin.H{"type": "integer", "format": "int64"},
			"latestEmailId":  gin.H{"anyOf": []gin.H{{"type": "integer", "format": "int64"}, {"type": "null"}}},
			"latestSubject":  gin.H{"anyOf": []gin.H{{"type": "string"}, {"type": "null"}}},
			"latestReceived": nullableDateTime,
		}},
		"CloudflareZone": gin.H{"type": "object", "properties": gin.H{
			"id":        gin.H{"type": "string"},
			"name":      gin.H{"type": "string"},
			"accountId": gin.H{"type": "string"},
			"selected":  gin.H{"type": "boolean"},
			"status":    gin.H{"type": "string"},
			"updatedAt": dateTime,
		}},
		"CloudflareStatus": gin.H{"type": "object", "properties": gin.H{
			"zoneId":              gin.H{"type": "string"},
			"zoneName":            gin.H{"type": "string"},
			"accountId":           gin.H{"type": "string"},
			"emailRoutingEnabled": gin.H{"type": "boolean"},
			"emailRoutingStatus":  gin.H{"type": "string"},
			"workerScriptName":    gin.H{"type": "string"},
			"catchAllEnabled":     gin.H{"type": "boolean"},
			"catchAllDestination": gin.H{"type": "string"},
		}},
		"CloudflareCredentialsRequest": gin.H{"type": "object", "properties": gin.H{
			"email":  gin.H{"type": "string"},
			"apiKey": gin.H{"type": "string"},
		}, "required": []string{"email", "apiKey"}},
		"ChangePasswordRequest": gin.H{"type": "object", "properties": gin.H{
			"oldPassword": gin.H{"type": "string"},
			"newPassword": gin.H{"type": "string"},
		}, "required": []string{"oldPassword", "newPassword"}},
		"IngestPayload": gin.H{"type": "object", "properties": gin.H{
			"provider":    gin.H{"type": "string"},
			"receivedAt":  dateTime,
			"messageId":   gin.H{"type": "string"},
			"mailFrom":    gin.H{"type": "string"},
			"rcptTo":      stringArray,
			"subject":     gin.H{"type": "string"},
			"text":        gin.H{"type": "string"},
			"html":        gin.H{"type": "string"},
			"headers":     gin.H{"type": "object", "additionalProperties": stringArray},
			"attachments": gin.H{"type": "array", "items": gin.H{"$ref": "#/components/schemas/Attachment"}},
			"rawSize":     gin.H{"type": "integer", "format": "int64"},
		}},
	}
}
