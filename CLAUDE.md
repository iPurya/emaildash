# CLAUDE.md

This file provides guidance to Claude Code when working in this repository.

## Common commands

### Backend
- Install deps: `cd backend && go mod tidy`
- Generate templ code: `cd backend && go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate`
- Run app: `cd backend && go run ./cmd/emaildash`
- Build all packages: `cd backend && go build ./...`
- Run all tests: `cd backend && go test ./...`

### Worker
- Install deps: `cd worker && npm install`
- Build bundle: `cd worker && npm run build`
- Run local Worker dev loop: `cd worker && npm run dev`

### Docker / full stack
- Start local stack: `docker compose -f deploy/docker-compose.yml up --build`
- Validate compose config: `docker compose -f deploy/docker-compose.yml config`

## High-level architecture

Repo is small monorepo with three parts:
- `backend/` Go app serving templ dashboard, REST API, auth, SQLite persistence, Cloudflare automation, and webhook ingest
- `worker/` Cloudflare Email Worker that parses inbound mail and POSTs to backend webhook
- `deploy/` Dockerfiles and compose for local/prod runs

### Request/data flow
1. Cloudflare Email Routing catch-all sends message to Worker.
2. Worker parses raw MIME with `postal-mime`, signs payload, and sends JSON to backend webhook.
3. Backend webhook verifies signature, stores email/recipients/attachments metadata in SQLite, and writes attachment content to disk.
4. Same Go app serves dashboard UI and REST API for browsing stored mail.

## Backend shape

Backend follows light clean-architecture split:
- `internal/domain/` shared data models
- `internal/usecase/` business flows
- `internal/adapters/` HTTP, SQLite, Cloudflare API, templ views
- `internal/platform/` config, crypto, signing helpers

### Important backend pieces

- `internal/adapters/http/router.go`
  - serves browser pages (`/`, `/setup`, `/login`, `/dashboard`)
  - serves inbox HTMX fragments under `/ui/...`
  - keeps REST API under `/api/...`
  - keeps webhook at `/api/ingest/cloudflare/email`

- `internal/adapters/http/handlers/pages.go`
  - page/HTMX handler layer for setup, login, dashboard, Cloudflare, and password flows

- `internal/adapters/http/views/`
  - `templ` templates and helpers for server-rendered UI

- `internal/adapters/sqlite/db.go`
  - repository layer for nearly all app state
  - owns setup state, user/session storage, encrypted secret storage, cached Cloudflare zones, emails, recipients, attachments, audit log

- `internal/usecase/cloudflare/service.go`
  - stores encrypted Cloudflare credentials
  - lists zones
  - provisions selected zone by enabling email routing, ensuring Worker subdomain, uploading Worker bundle from disk, pushing Worker secrets, binding catch-all rule
  - depends on `EMAILDASH_WORKER_BUNDLE` and `EMAILDASH_PUBLIC_BASE_URL`

## Worker shape

`worker/src/index.ts` is the single important file.
- Cloudflare `email()` handler only
- parses `message.raw`
- extracts subject/text/html/headers/attachments
- computes HMAC signature with `EMAILDASH_WEBHOOK_SECRET`
- sends payload to `EMAILDASH_WEBHOOK_URL`

Worker build output goes to `worker/dist/index.js`. Backend Cloudflare provisioning reads that built bundle directly from disk.

## Runtime/data assumptions

- SQLite DB, attachments, and master key live under `data/` by default.
- Backend default public URL is `http://localhost:8080`; real Cloudflare end-to-end testing needs public HTTPS URL.
- App runtime is one Go application container serving UI + API + webhook. Cloudflare Worker remains external because Cloudflare runs it.

## Current project-specific cautions

- Cloudflare adapter still needs real staging validation against live Cloudflare account/domain. If Cloudflare API requests fail, check request shape first before changing usecase flow.
- Attachment persistence currently writes `Attachment.Content` string directly to disk. If attachment corruption appears, inspect Worker encoding and ingest decode path first.
- No dedicated lint script exists yet. Do not document or rely on nonexistent lint command.
