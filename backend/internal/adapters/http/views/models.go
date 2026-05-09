package views

import "github.com/purya/emaildash/backend/internal/domain"

type SetupPageData struct {
	Error string
}

type LoginPageData struct {
	Error string
}

type DocsEndpoint struct {
	Method      string
	Path        string
	Description string
	Auth        string
	Request     string
	Response    string
	AgentUse    string
}

type DocsStep struct {
	Title string
	Body  string
}

type DocsField struct {
	Name        string
	Type        string
	Description string
}

type DocsSchema struct {
	Name        string
	Description string
	Fields      []DocsField
}

type APIDocsData struct {
	BaseURL          string
	OpenAPIURL       string
	AgentMarkdownURL string
	SampleAuth       string
	AgentPrompt      string
	PythonSDKGuide   string
	AgentRules       []string
	Workflow         []DocsStep
	Endpoints        []DocsEndpoint
	Schemas          []DocsSchema
}

type DashboardData struct {
	Title           string
	ActiveTab       string
	Error           string
	Notice          string
	APIKey          string
	ActiveRecipient string
	SelectedEmailID int64
	Recipients      []domain.RecipientSummary
	Emails          []domain.Email
	ActiveEmail     *domain.Email
	Zones           []domain.CloudflareZone
	Status          *domain.CloudflareStatus
}
