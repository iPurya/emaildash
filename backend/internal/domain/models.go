package domain

import "time"

type SetupStatus struct {
	Initialized bool `json:"initialized"`
}

type User struct {
	ID             int64     `json:"id"`
	PasswordHash   string    `json:"-"`
	PasswordParams string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Session struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	TokenHash  string     `json:"-"`
	CSRFToken  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Priority int    `json:"priority,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
}

type SecretRecord struct {
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CloudflareCredentials struct {
	Email  string `json:"email"`
	APIKey string `json:"apiKey"`
}

type CloudflareZone struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	AccountID string    `json:"accountId"`
	Selected  bool      `json:"selected"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CloudflareStatus struct {
	ZoneID              string `json:"zoneId"`
	ZoneName            string `json:"zoneName"`
	AccountID           string `json:"accountId"`
	EmailRoutingEnabled bool   `json:"emailRoutingEnabled"`
	EmailRoutingStatus  string `json:"emailRoutingStatus"`
	WorkerScriptName    string `json:"workerScriptName"`
	CatchAllEnabled     bool   `json:"catchAllEnabled"`
	CatchAllDestination string `json:"catchAllDestination"`
}

type EmailListFilter struct {
	Recipient  string `json:"recipient,omitempty"`
	FromMail   string `json:"fromMail,omitempty"`
	ToMail     string `json:"toMail,omitempty"`
	UnreadOnly bool   `json:"unreadOnly,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type Attachment struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	StoragePath string `json:"storagePath,omitempty"`
	Content     string `json:"content,omitempty"`
}

type Email struct {
	ID                int64               `json:"id"`
	Provider          string              `json:"provider"`
	ProviderMessageID string              `json:"providerMessageId"`
	MailFrom          string              `json:"mailFrom"`
	Recipients        []string            `json:"recipients"`
	Subject           string              `json:"subject"`
	TextBody          string              `json:"textBody"`
	HTMLBody          string              `json:"htmlBody"`
	Headers           map[string][]string `json:"headers"`
	RawSize           int64               `json:"rawSize"`
	ReadAt            *time.Time          `json:"readAt,omitempty"`
	ReceivedAt        time.Time           `json:"receivedAt"`
	CreatedAt         time.Time           `json:"createdAt"`
	Attachments       []Attachment        `json:"attachments"`
}

type RecipientSummary struct {
	Address        string     `json:"address"`
	Count          int64      `json:"count"`
	UnreadCount    int64      `json:"unreadCount"`
	LatestEmailID  *int64     `json:"latestEmailId,omitempty"`
	LatestSubject  *string    `json:"latestSubject,omitempty"`
	LatestReceived *time.Time `json:"latestReceived,omitempty"`
}

type IngestPayload struct {
	Provider    string              `json:"provider"`
	ReceivedAt  time.Time           `json:"receivedAt"`
	MessageID   string              `json:"messageId"`
	MailFrom    string              `json:"mailFrom"`
	RcptTo      []string            `json:"rcptTo"`
	Subject     string              `json:"subject"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	Headers     map[string][]string `json:"headers"`
	Attachments []Attachment        `json:"attachments"`
	RawSize     int64               `json:"rawSize"`
}
