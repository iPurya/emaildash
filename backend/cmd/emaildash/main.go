package main

import (
	"log"
	"os"
	"time"

	cloudflareadapter "github.com/purya/emaildash/backend/internal/adapters/cloudflare"
	httpadapter "github.com/purya/emaildash/backend/internal/adapters/http"
	"github.com/purya/emaildash/backend/internal/adapters/http/handlers"
	"github.com/purya/emaildash/backend/internal/adapters/sqlite"
	"github.com/purya/emaildash/backend/internal/platform/config"
	appcrypto "github.com/purya/emaildash/backend/internal/platform/crypto"
	"github.com/purya/emaildash/backend/internal/platform/signing"
	authusecase "github.com/purya/emaildash/backend/internal/usecase/auth"
	cloudflareusecase "github.com/purya/emaildash/backend/internal/usecase/cloudflare"
	ingestusecase "github.com/purya/emaildash/backend/internal/usecase/ingest"
	inboxusecase "github.com/purya/emaildash/backend/internal/usecase/inbox"
	setupusecase "github.com/purya/emaildash/backend/internal/usecase/setup"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.AttachmentDir, 0o755); err != nil {
		log.Fatal(err)
	}
	store, err := sqlite.NewStore(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	hasher := appcrypto.NewPasswordHasher()
	sealer, err := appcrypto.NewSecretSealer(cfg.MasterKeyPath)
	if err != nil {
		log.Fatal(err)
	}
	signer := signing.NewHMACSigner()
	cloudflareClient := cloudflareadapter.NewClient()

	setupService := setupusecase.NewService(store, hasher)
	authService := authusecase.NewService(store, hasher, time.Duration(cfg.SessionTTLHours)*time.Hour)
	cloudflareService := cloudflareusecase.NewService(store, cloudflareClient, sealer, cfg.WorkerScriptName, cfg.WorkerSubdomain, cfg.WorkerBundlePath, cfg.PublicBaseURL)
	ingestService := ingestusecase.NewService(store, sealer, signer, cfg.AttachmentDir)
	inboxService := inboxusecase.NewService(store)

	router := httpadapter.NewRouter(cfg, httpadapter.Services{
		Setup:      handlers.NewSetupHandler(setupService),
		Auth:       handlers.NewAuthHandler(authService, cfg.CookieName),
		Cloudflare: handlers.NewCloudflareHandler(cloudflareService),
		Ingest:     handlers.NewIngestHandler(ingestService),
		Emails:     handlers.NewEmailsHandler(inboxService),
		Pages:      handlers.NewPagesHandler(setupService, authService, cloudflareService, inboxService, cfg.CookieName),
		AuthSvc:    authService,
	})

	if err := router.Run(cfg.Address()); err != nil {
		log.Fatal(err)
	}
}
