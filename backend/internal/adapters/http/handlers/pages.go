package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	ui "github.com/purya/emaildash/backend/internal/adapters/http/views"
	"github.com/purya/emaildash/backend/internal/domain"
	authusecase "github.com/purya/emaildash/backend/internal/usecase/auth"
	cloudflareusecase "github.com/purya/emaildash/backend/internal/usecase/cloudflare"
	inboxusecase "github.com/purya/emaildash/backend/internal/usecase/inbox"
	setupusecase "github.com/purya/emaildash/backend/internal/usecase/setup"
)

type PagesHandler struct {
	setup      setupusecase.Service
	auth       authusecase.Service
	cloudflare cloudflareusecase.Service
	inbox      inboxusecase.Service
	cookieName string
}

func NewPagesHandler(setup setupusecase.Service, auth authusecase.Service, cloudflare cloudflareusecase.Service, inbox inboxusecase.Service, cookieName string) PagesHandler {
	return PagesHandler{setup: setup, auth: auth, cloudflare: cloudflare, inbox: inbox, cookieName: cookieName}
}

func (h PagesHandler) Root(c *gin.Context) {
	status, err := h.setup.Status(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if !status.Initialized {
		c.Redirect(http.StatusFound, "/setup")
		return
	}
	token, err := c.Cookie(h.cookieName)
	if err != nil || token == "" {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	if _, err := h.auth.Authenticate(c.Request.Context(), token); err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	c.Redirect(http.StatusFound, "/dashboard?tab=inbox")
}

func (h PagesHandler) APIDocsPage(c *gin.Context) {
	baseURL := c.Request.URL.Scheme + "://" + c.Request.Host
	if c.Request.TLS == nil {
		if c.GetHeader("X-Forwarded-Proto") == "https" {
			baseURL = "https://" + c.Request.Host
		} else {
			baseURL = "http://" + c.Request.Host
		}
	}
	data := ui.APIDocsData{
		BaseURL:    baseURL,
		SampleAuth: "api_key=YOUR_API_KEY",
		Endpoints: []ui.DocsEndpoint{
			{Method: "GET", Path: "/api/setup/status", Description: "Check whether initial setup has already completed.", Auth: "public", Example: fmt.Sprintf("curl %s/api/setup/status", baseURL), Response: `{"initialized":true}`},
			{Method: "POST", Path: "/api/auth/login", Description: "Create browser session cookie using dashboard password.", Auth: "public", Example: fmt.Sprintf(`curl -c cookies.txt -H "Content-Type: application/json" -d "{\"password\":\"YOUR_PASSWORD\"}" %s/api/auth/login`, baseURL), Response: `{"csrfToken":"...","expiresAt":"..."}`},
			{Method: "GET", Path: "/api/auth/me", Description: "Inspect current cookie-authenticated session.", Auth: "cookie", Example: fmt.Sprintf("curl -b cookies.txt %s/api/auth/me", baseURL), Response: `{"authenticated":true,"expiresAt":"...","csrfToken":"..."}`},
			{Method: "GET", Path: "/api/emails", Description: "List emails. Optional filters: recipient, to_mail, from_mail, unread, limit, api_key.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl "%s/api/emails?api_key=YOUR_API_KEY&to_mail=test1@purya.me&from_mail=alice@example.com"`, baseURL), Response: `{"emails":[{"id":1,"mailFrom":"alice@example.com","recipients":["test1@purya.me"],"subject":"hello"}]}`},
			{Method: "GET", Path: "/api/emails/:id", Description: "Fetch one full email by ID.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl "%s/api/emails/1?api_key=YOUR_API_KEY"`, baseURL), Response: `{"id":1,"mailFrom":"alice@example.com","recipients":["test1@purya.me"],"subject":"hello","textBody":"...","attachments":[]}`},
			{Method: "GET", Path: "/api/recipients", Description: "List grouped recipient summaries used by dashboard sidebar.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl "%s/api/recipients?api_key=YOUR_API_KEY"`, baseURL), Response: `{"recipients":[{"address":"test1@purya.me","count":2,"unreadCount":1}]}`},
			{Method: "PATCH", Path: "/api/emails/:id/read", Description: "Mark one email as read.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl -X PATCH "%s/api/emails/1/read?api_key=YOUR_API_KEY"`, baseURL), Response: `HTTP 204 No Content`},
			{Method: "GET", Path: "/api/cloudflare/zones", Description: "Return cached Cloudflare zones after credentials are saved.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl "%s/api/cloudflare/zones?api_key=YOUR_API_KEY"`, baseURL), Response: `{"zones":[{"id":"...","name":"purya.dev","selected":true}]}`},
			{Method: "GET", Path: "/api/cloudflare/status", Description: "Return status for the selected Cloudflare zone.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl "%s/api/cloudflare/status?api_key=YOUR_API_KEY"`, baseURL), Response: `{"zoneName":"purya.dev","emailRoutingEnabled":true,"catchAllEnabled":true}`},
			{Method: "POST", Path: "/api/cloudflare/credentials", Description: "Save Cloudflare account email and Global API key.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl -H "Content-Type: application/json" -d "{\"email\":\"name@example.com\",\"apiKey\":\"YOUR_CF_KEY\"}" "%s/api/cloudflare/credentials?api_key=YOUR_API_KEY"`, baseURL), Response: `{"zones":[{"id":"...","name":"purya.dev"}]}`},
			{Method: "POST", Path: "/api/cloudflare/zones/:zoneId/provision", Description: "Provision Cloudflare routing and worker for one zone.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl -X POST "%s/api/cloudflare/zones/ZONE_ID/provision?api_key=YOUR_API_KEY"`, baseURL), Response: `{"zoneName":"purya.dev","workerScriptName":"emaildash-ingest"}`},
			{Method: "POST", Path: "/api/settings/password", Description: "Change password and rotate API key. Response returns the new API key.", Auth: "cookie or api_key", Example: fmt.Sprintf(`curl -H "Content-Type: application/json" -d "{\"oldPassword\":\"old\",\"newPassword\":\"new\"}" "%s/api/settings/password?api_key=YOUR_API_KEY"`, baseURL), Response: `{"apiKey":"NEW_API_KEY"}`},
			{Method: "POST", Path: "/api/ingest/cloudflare/email", Description: "Webhook endpoint used by the Cloudflare Worker. Not for manual dashboard/API use.", Auth: "signed webhook", Example: `Handled by worker only.`, Response: `HTTP 201 Created`},
		},
	}
	ui.Render(c.Request.Context(), c.Writer, ui.APIDocsPage(data)).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) SetupPage(c *gin.Context) {
	status, err := h.setup.Status(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if status.Initialized {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	ui.Render(c.Request.Context(), c.Writer, ui.SetupPage(ui.SetupPageData{Error: c.Query("error")})).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) SetupSubmit(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		h.redirect(c, "/setup", url.Values{"error": {"password required"}})
		return
	}
	if err := h.setup.Initialize(c.Request.Context(), password); err != nil {
		h.redirect(c, "/setup", url.Values{"error": {err.Error()}})
		return
	}
	if _, err := h.auth.EnsureAPIKey(c.Request.Context()); err != nil {
		h.redirect(c, "/setup", url.Values{"error": {err.Error()}})
		return
	}
	token, session, err := h.auth.Login(c.Request.Context(), password)
	if err != nil {
		h.redirect(c, "/setup", url.Values{"error": {err.Error()}})
		return
	}
	secureCookie := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, token, int(time.Until(session.ExpiresAt).Seconds()), "/", "", secureCookie, true)
	c.Redirect(http.StatusFound, "/dashboard?tab=cloudflare")
}

func (h PagesHandler) LoginPage(c *gin.Context) {
	ui.Render(c.Request.Context(), c.Writer, ui.LoginPage(ui.LoginPageData{Error: c.Query("error")})).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) LoginSubmit(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		h.redirect(c, "/login", url.Values{"error": {"password required"}})
		return
	}
	token, session, err := h.auth.Login(c.Request.Context(), password)
	if err != nil {
		h.redirect(c, "/login", url.Values{"error": {err.Error()}})
		return
	}
	secureCookie := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, token, int(time.Until(session.ExpiresAt).Seconds()), "/", "", secureCookie, true)
	c.Redirect(http.StatusFound, "/dashboard?tab=inbox")
}

func (h PagesHandler) LogoutSubmit(c *gin.Context) {
	token, _ := c.Cookie(h.cookieName)
	_ = h.auth.Logout(c.Request.Context(), token)
	secureCookie := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie(h.cookieName, "", -1, "/", "", secureCookie, true)
	c.Redirect(http.StatusFound, "/login")
}

func (h PagesHandler) DashboardPage(c *gin.Context) {
	data, err := h.dashboardData(c)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	ui.Render(c.Request.Context(), c.Writer, ui.DashboardPage(data)).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) RecipientsFragment(c *gin.Context) {
	data, err := h.dashboardData(c)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	ui.Render(c.Request.Context(), c.Writer, ui.RecipientsPartial(data)).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) EmailsFragment(c *gin.Context) {
	data, err := h.dashboardData(c)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	ui.Render(c.Request.Context(), c.Writer, ui.EmailsPartial(data)).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) ViewerFragment(c *gin.Context) {
	data, err := h.dashboardData(c)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if data.ActiveEmail != nil && data.ActiveEmail.ReadAt == nil {
		_ = h.inbox.MarkRead(c.Request.Context(), data.ActiveEmail.ID)
		now := time.Now().UTC()
		data.ActiveEmail.ReadAt = &now
	}
	ui.Render(c.Request.Context(), c.Writer, ui.ViewerPartial(data)).ServeHTTP(c.Writer, c.Request)
}

func (h PagesHandler) PasswordSubmit(c *gin.Context) {
	oldPassword := c.PostForm("oldPassword")
	newPassword := c.PostForm("newPassword")
	apiKey, err := h.auth.ChangePassword(c.Request.Context(), oldPassword, newPassword)
	if err != nil {
		h.redirectDashboard(c, "password", "", 0, url.Values{"error": {err.Error()}})
		return
	}
	h.redirectDashboard(c, "password", "", 0, url.Values{"updated": {"1"}, "api_key": {apiKey}})
}

func (h PagesHandler) CloudflareCredentialsSubmit(c *gin.Context) {
	_, err := h.cloudflare.SaveCredentials(c.Request.Context(), domain.CloudflareCredentials{
		Email:  c.PostForm("email"),
		APIKey: c.PostForm("apiKey"),
	})
	if err != nil {
		h.redirectDashboard(c, "cloudflare", "", 0, url.Values{"error": {err.Error()}})
		return
	}
	h.redirectDashboard(c, "cloudflare", "", 0, url.Values{"saved": {"1"}})
}

func (h PagesHandler) CloudflareProvisionSubmit(c *gin.Context) {
	zoneID := c.PostForm("zoneId")
	if zoneID == "" {
		h.redirectDashboard(c, "cloudflare", "", 0, url.Values{"error": {"zone required"}})
		return
	}
	if _, err := h.cloudflare.ProvisionZone(c.Request.Context(), zoneID); err != nil {
		h.redirectDashboard(c, "cloudflare", "", 0, url.Values{"error": {err.Error()}})
		return
	}
	h.redirectDashboard(c, "cloudflare", "", 0, url.Values{"provisioned": {"1"}})
}

func (h PagesHandler) dashboardData(c *gin.Context) (ui.DashboardData, error) {
	data := ui.DashboardData{
		Title:           "Dashboard",
		ActiveTab:       c.DefaultQuery("tab", "inbox"),
		Error:           c.Query("error"),
		Notice:          h.notice(c),
		APIKey:          c.Query("api_key"),
		ActiveRecipient: c.Query("recipient"),
	}
	selectedID, _ := strconv.ParseInt(c.Query("email"), 10, 64)
	data.SelectedEmailID = selectedID

	switch data.ActiveTab {
	case "cloudflare":
		zones, err := h.cloudflare.ListCachedZones(c.Request.Context())
		if err == nil {
			data.Zones = zones
		}
		if status, err := h.cloudflare.Status(c.Request.Context()); err == nil {
			data.Status = &status
		}
	case "password":
		if data.APIKey == "" {
			apiKey, err := h.auth.EnsureAPIKey(c.Request.Context())
			if err == nil {
				data.APIKey = apiKey
			}
		}
	default:
		data.ActiveTab = "inbox"
		recipients, err := h.inbox.ListRecipients(c.Request.Context())
		if err != nil {
			return data, err
		}
		data.Recipients = recipients
		emails, err := h.inbox.ListEmails(c.Request.Context(), domain.EmailListFilter{Recipient: data.ActiveRecipient, ToMail: data.ActiveRecipient, Limit: 50})
		if err != nil {
			return data, err
		}
		data.Emails = emails
		if data.SelectedEmailID == 0 && len(emails) > 0 {
			data.SelectedEmailID = emails[0].ID
		}
		for i := range emails {
			if emails[i].ID == data.SelectedEmailID {
				data.ActiveEmail = &emails[i]
				break
			}
		}
	}

	return data, nil
}

func (h PagesHandler) notice(c *gin.Context) string {
	switch {
	case c.Query("updated") == "1":
		return "Password updated and API key rotated."
	case c.Query("saved") == "1":
		return "Cloudflare credentials saved."
	case c.Query("provisioned") == "1":
		return "Cloudflare zone provisioned."
	default:
		return ""
	}
}

func (h PagesHandler) redirectDashboard(c *gin.Context, tab, recipient string, emailID int64, extra url.Values) {
	values := extra
	if values == nil {
		values = url.Values{}
	}
	values.Set("tab", tab)
	if recipient != "" {
		values.Set("recipient", recipient)
	}
	if emailID != 0 {
		values.Set("email", strconv.FormatInt(emailID, 10))
	}
	c.Redirect(http.StatusFound, "/dashboard?"+values.Encode())
}

func (h PagesHandler) redirect(c *gin.Context, path string, values url.Values) {
	if len(values) == 0 {
		c.Redirect(http.StatusFound, path)
		return
	}
	c.Redirect(http.StatusFound, path+"?"+values.Encode())
}
