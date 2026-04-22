# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

### Backend
- Install deps: `cd backend && go mod tidy`
- Run app: `cd backend && go run ./cmd/emaildash`
- Build all packages: `cd backend && go build ./...`
- Run all tests: `cd backend && go test ./...`
- Run single package tests: `cd backend && go test ./internal/usecase/auth`
- Run single test: `cd backend && go test ./internal/usecase/auth -run TestName`

### Frontend
- Install deps: `cd frontend && npm install`
- Run dev server: `cd frontend && npm run dev`
- Build: `cd frontend && npm run build`

### Worker
- Install deps: `cd worker && npm install`
- Build bundle: `cd worker && npm run build`
- Run local Worker dev loop: `cd worker && npm run dev`

### Docker / full stack
- Start local stack: `docker compose -f deploy/docker-compose.yml up --build`
- Validate compose config: `docker compose -f deploy/docker-compose.yml config`

### Linting
- No dedicated lint command configured currently in repo.

## High-level architecture

Repo is small monorepo with four parts:
- `backend/` Go API, auth, SQLite persistence, Cloudflare automation, webhook ingest
- `frontend/` React dashboard and setup/login flows
- `worker/` Cloudflare Email Worker that parses inbound mail and POSTs to backend
- `deploy/` Dockerfiles and compose for local/full-stack runs

### Request/data flow
1. Cloudflare Email Routing catch-all sends message to Worker.
2. Worker parses raw MIME with `postal-mime`, builds JSON payload, signs payload with HMAC, sends to backend webhook.
3. Backend webhook verifies signature, normalizes payload, stores email/recipients/attachments metadata in SQLite, writes attachment content to disk.
4. React app reads grouped recipients and email detail via backend REST API.

## Backend shape

Backend follows light clean-architecture split:
- `internal/domain/` shared data models
- `internal/usecase/` business flows
- `internal/adapters/` HTTP, SQLite, Cloudflare API
- `internal/platform/` config, crypto, signing helpers

`backend/cmd/emaildash/main.go` is composition root. It wires:
- config loader
- SQLite store
- Argon2 password hasher
- AES-GCM secret sealer
- HMAC signer
- Cloudflare HTTP client
- setup/auth/cloudflare/ingest/inbox services
- Gin router

### Important backend pieces

- `internal/adapters/http/router.go`
  - splits public routes from authenticated routes
  - public: setup status/init, login/logout, ingest webhook
  - authenticated: auth/me, Cloudflare, inbox, settings
  - CSRF middleware wraps authenticated write routes
  - also serves embedded static fallback from `internal/adapters/http/static/`

- `internal/adapters/sqlite/db.go`
  - acts as repository layer for nearly all app state
  - applies embedded SQL migrations on startup
  - owns setup state, user/session storage, encrypted secret storage, cached Cloudflare zones, emails, recipients, attachments, audit log
  - email queries return normalized records and join recipient/attachment tables

- `internal/usecase/cloudflare/service.go`
  - orchestration layer for Cloudflare setup
  - stores encrypted Cloudflare credentials
  - lists zones
  - provisions selected zone by enabling email routing, ensuring Worker subdomain, uploading Worker bundle from disk, pushing Worker secrets, binding catch-all rule
  - depends on `EMAILDASH_WORKER_BUNDLE` and `EMAILDASH_PUBLIC_BASE_URL` being correct

- `internal/usecase/ingest/service.go`
  - verifies signed webhook requests
  - fills fallback message ID if needed
  - writes attachments under attachment dir
  - converts payload into `domain.Email` before insert

- `internal/platform/crypto/`
  - `password.go`: Argon2id password hashing and verification
  - `secrets.go`: AES-GCM encryption using local master key file

- `internal/platform/signing/hmac.go`
  - worker/backend shared HMAC verification format: `timestamp.body`

## Frontend shape

Frontend is Vite + React Router + TanStack Query.

`frontend/src/app/router.tsx` is entry point for app state machine:
- calls setup-status query
- calls auth/me query
- decides between `SetupWizard`, `Login`, or authenticated shell
- stores CSRF token in module-level API state via `setCSRFToken`

Important consequence: authenticated write requests rely on frontend having called auth/me or login first so `frontend/src/lib/api.ts` can attach CSRF header automatically.

Inbox UI is three-pane layout:
- recipient groups
- email list
- email viewer

Settings routes are nested under `/settings`.

## Worker shape

`worker/src/index.ts` is single important file.
- Cloudflare `email()` handler only
- parses `message.raw`
- extracts subject/text/html/headers/attachments
- computes HMAC signature with `EMAILDASH_WEBHOOK_SECRET`
- sends payload to `EMAILDASH_WEBHOOK_URL`

Worker build output goes to `worker/dist/index.js`. Backend Cloudflare provisioning reads that built bundle directly from disk.

## Runtime/data assumptions

- SQLite DB, attachments, and master key live under `data/` by default.
- Backend default public URL is `http://localhost:8080`; real Cloudflare end-to-end testing needs public HTTPS URL.
- Backend serves placeholder embedded static page unless frontend build artifacts are mounted/copied into runtime path.

## Current project-specific cautions

- Cloudflare adapter is scaffolded but still needs real staging validation against live Cloudflare account/domain. If Cloudflare API requests fail, check request shape first before changing usecase flow.
- Attachment persistence currently writes `Attachment.Content` string directly to disk. If attachment corruption appears, inspect Worker encoding and ingest decode path first.
- No dedicated lint script exists yet. Do not document or rely on nonexistent lint command.
