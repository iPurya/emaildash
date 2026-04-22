package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port              string
	DataDir           string
	DBPath            string
	AttachmentDir     string
	MasterKeyPath     string
	PublicBaseURL     string
	CookieName        string
	CSRFHeader        string
	WorkerScriptName  string
	WorkerSubdomain   string
	FrontendDistDir   string
	WorkerBundlePath  string
	SessionTTLHours   int
	AllowedOrigin     string
}

func Load() Config {
	dataDir := envOrDefault("EMAILDASH_DATA_DIR", filepath.Clean(filepath.Join(".", "..", "data")))
	frontendDist := envOrDefault("EMAILDASH_FRONTEND_DIST", filepath.Clean(filepath.Join("..", "frontend", "dist")))
	workerBundle := envOrDefault("EMAILDASH_WORKER_BUNDLE", filepath.Clean(filepath.Join("..", "worker", "dist", "index.js")))
	cfg := Config{
		Port:             envOrDefault("PORT", "8080"),
		DataDir:          dataDir,
		DBPath:           envOrDefault("EMAILDASH_DB_PATH", filepath.Join(dataDir, "emaildash.db")),
		AttachmentDir:    envOrDefault("EMAILDASH_ATTACHMENT_DIR", filepath.Join(dataDir, "attachments")),
		MasterKeyPath:    envOrDefault("EMAILDASH_MASTER_KEY_PATH", filepath.Join(dataDir, ".masterkey")),
		PublicBaseURL:    envOrDefault("EMAILDASH_PUBLIC_BASE_URL", "http://localhost:8080"),
		CookieName:       envOrDefault("EMAILDASH_COOKIE_NAME", "emaildash_session"),
		CSRFHeader:       envOrDefault("EMAILDASH_CSRF_HEADER", "X-CSRF-Token"),
		WorkerScriptName: envOrDefault("EMAILDASH_WORKER_SCRIPT_NAME", "emaildash-ingest"),
		WorkerSubdomain:  envOrDefault("EMAILDASH_WORKER_SUBDOMAIN", "emaildash-receiver"),
		FrontendDistDir:  frontendDist,
		WorkerBundlePath: workerBundle,
		SessionTTLHours:  envOrDefaultInt("EMAILDASH_SESSION_TTL_HOURS", 24*14),
		AllowedOrigin:    envOrDefault("EMAILDASH_ALLOWED_ORIGIN", "http://localhost:5173"),
	}
	return cfg
}

func (c Config) Address() string {
	return fmt.Sprintf(":%s", c.Port)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
