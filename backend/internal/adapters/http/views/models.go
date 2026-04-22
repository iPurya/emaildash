package views

import "github.com/purya/emaildash/backend/internal/domain"

type SetupPageData struct {
	Error string
}

type LoginPageData struct {
	Error string
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
