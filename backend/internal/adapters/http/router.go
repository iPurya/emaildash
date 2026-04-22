package httpadapter

import (
	"context"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/adapters/http/handlers"
	"github.com/purya/emaildash/backend/internal/adapters/http/middleware"
	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/platform/config"
)

type Services struct {
	Setup      handlers.SetupHandler
	Auth       handlers.AuthHandler
	Cloudflare handlers.CloudflareHandler
	Ingest     handlers.IngestHandler
	Emails     handlers.EmailsHandler
	Pages      handlers.PagesHandler
	AuthSvc    interface {
		Authenticate(ctx context.Context, token string) (domain.Session, error)
		AuthenticateAPIKey(ctx context.Context, apiKey string) error
	}
}

func NewRouter(cfg config.Config, services Services) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigin, cfg.PublicBaseURL},
		AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	router.GET("/", services.Pages.Root)
	router.GET("/setup", services.Pages.SetupPage)
	router.POST("/setup", services.Pages.SetupSubmit)
	router.GET("/login", services.Pages.LoginPage)
	router.POST("/login", services.Pages.LoginSubmit)

	pages := router.Group("")
	pages.Use(middleware.RequirePageAuth(cfg.CookieName, services.AuthSvc))
	pages.POST("/logout", services.Pages.LogoutSubmit)
	pages.GET("/dashboard", services.Pages.DashboardPage)
	pages.GET("/ui/inbox/recipients", services.Pages.RecipientsFragment)
	pages.GET("/ui/inbox/emails", services.Pages.EmailsFragment)
	pages.GET("/ui/inbox/viewer", services.Pages.ViewerFragment)
	pages.POST("/dashboard/password", services.Pages.PasswordSubmit)
	pages.POST("/dashboard/cloudflare/credentials", services.Pages.CloudflareCredentialsSubmit)
	pages.POST("/dashboard/cloudflare/provision", services.Pages.CloudflareProvisionSubmit)

	api := router.Group("/api")
	api.GET("/setup/status", services.Setup.Status)
	api.POST("/setup/initialize", services.Setup.Initialize)
	api.POST("/auth/login", services.Auth.Login)
	api.POST("/auth/logout", services.Auth.Logout)
	api.POST("/ingest/cloudflare/email", services.Ingest.Receive)

	authed := api.Group("")
	authed.Use(middleware.RequireAuth(cfg.CookieName, services.AuthSvc))
	authed.GET("/auth/me", services.Auth.Me)
	authed.GET("/cloudflare/zones", services.Cloudflare.ListZones)
	authed.GET("/cloudflare/status", services.Cloudflare.Status)
	authed.GET("/emails", services.Emails.List)
	authed.GET("/emails/:id", services.Emails.Get)
	authed.GET("/recipients", services.Emails.ListRecipients)
	authed.POST("/settings/password", services.Auth.ChangePassword)
	authed.POST("/cloudflare/credentials", services.Cloudflare.SaveCredentials)
	authed.POST("/cloudflare/zones/:zoneId/provision", services.Cloudflare.Provision)
	authed.PATCH("/emails/:id/read", services.Emails.MarkRead)

	return router
}
