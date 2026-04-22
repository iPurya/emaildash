package handlers

import (
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
