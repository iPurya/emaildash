package httpadapter

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/purya/emaildash/backend/internal/adapters/http/handlers"
	"github.com/purya/emaildash/backend/internal/adapters/http/middleware"
	"github.com/purya/emaildash/backend/internal/domain"
	"github.com/purya/emaildash/backend/internal/platform/config"
)

//go:embed static/*
var embeddedStatic embed.FS

type Services struct {
	Setup      handlers.SetupHandler
	Auth       handlers.AuthHandler
	Cloudflare handlers.CloudflareHandler
	Ingest     handlers.IngestHandler
	Emails     handlers.EmailsHandler
	AuthSvc    interface{ Authenticate(ctx context.Context, token string) (domain.Session, error) }
}

func NewRouter(cfg config.Config, services Services) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigin, cfg.PublicBaseURL},
		AllowMethods:     []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", cfg.CSRFHeader},
		AllowCredentials: true,
	}))

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
	authed.Use(middleware.RequireCSRF(cfg.CSRFHeader))
	authed.POST("/settings/password", services.Auth.ChangePassword)
	authed.POST("/cloudflare/credentials", services.Cloudflare.SaveCredentials)
	authed.POST("/cloudflare/zones/:zoneId/provision", services.Cloudflare.Provision)
	authed.PATCH("/emails/:id/read", services.Emails.MarkRead)

	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err == nil {
		router.GET("/", func(c *gin.Context) {
			c.FileFromFS("index.html", http.FS(staticFS))
		})
	}
	return router
}
