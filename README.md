# Emaildash

Modern inbound email dashboard built with Go, React, SQLite, and Cloudflare Email Routing.

Goal: minimal setup, single-user auth, automatic Cloudflare domain hookup, catch-all email ingest, grouped inbox by recipient, and REST API access to stored messages.

## Current status

Project scaffold is in place and builds successfully:
- Go backend builds and tests pass
- React frontend builds
- Cloudflare Worker builds
- Docker Compose config validates

Current implementation is working scaffold, not finished production release yet. Important gaps still exist:
- setup wizard UX still needs one smooth end-to-end auth + Cloudflare onboarding pass
- live Cloudflare API calls need staging validation against real account/domain
- attachment persistence path should be upgraded from current string-write scaffold to proper binary/base64 decode flow
- backend serves placeholder embedded static file unless frontend dist is mounted or copied into runtime image
- no automated unit/integration test coverage written yet beyond compile/test smoke

## Stack

- Backend: Go + Gin
- Database: SQLite via `modernc.org/sqlite`
- Frontend: React + Vite + Tailwind
- Worker: TypeScript + Cloudflare Email Worker + `postal-mime`
- Dev orchestration: Docker Compose

## Architecture

```text
Cloudflare Email Routing
  -> Catch-all rule
  -> Cloudflare Worker (`worker/src/index.ts`)
  -> Signed POST webhook
  -> Go backend (`/api/ingest/cloudflare/email`)
  -> SQLite + attachment storage
  -> React dashboard + REST API
```

### Main directories

```text
backend/   Go server, auth, SQLite, Cloudflare automation, ingest API
frontend/  React dashboard UI
worker/    Cloudflare Email Worker
deploy/    Dockerfiles and docker-compose.yml
data/      SQLite DB, secrets, attachments at runtime
PLAN.md    Original implementation plan
```

## Features in scaffold

### Backend
- single-user setup and password auth
- session cookie auth
- CSRF check on authenticated write endpoints
- encrypted secret storage for Cloudflare credentials and webhook secret
- SQLite schema and migration bootstrap
- inbox list/detail/read endpoints
- grouped recipients endpoint
- Cloudflare automation service boundary and HTTP adapter scaffold
- signed ingest webhook

### Frontend
- setup page
- login page
- grouped inbox layout
- settings pages for password and Cloudflare
- typed API client

### Worker
- parses inbound email with `postal-mime`
- extracts sender, recipient, subject, text, HTML, headers, attachments
- signs webhook payload with HMAC SHA-256

## API overview

### Setup and auth
- `GET /api/setup/status`
- `POST /api/setup/initialize`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/settings/password`

### Cloudflare
- `POST /api/cloudflare/credentials`
- `GET /api/cloudflare/zones`
- `POST /api/cloudflare/zones/:zoneId/provision`
- `GET /api/cloudflare/status`

### Inbox
- `GET /api/emails`
- `GET /api/emails/:id`
- `PATCH /api/emails/:id/read`
- `GET /api/recipients`

### Ingest
- `POST /api/ingest/cloudflare/email`

## Local setup

### Prerequisites

Install one of these flows:

#### Docker-first
- Docker Desktop
- Docker Compose

#### Native
- Go 1.24+
- Node.js 22+
- npm 10+

Cloudflare requirements for real end-to-end test:
- Cloudflare-managed domain
- Cloudflare account email
- Cloudflare Global API Key
- public HTTPS URL for backend webhook

## Quick start with Docker

From repo root:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Services:
- frontend: `http://localhost:5173`
- backend API: `http://localhost:8080`

Mounted runtime data:
- `./data/emaildash.db`
- `./data/attachments/`
- `./data/.masterkey`

## Native development

### 1. Backend

```bash
cd backend
go mod tidy
go run ./cmd/emaildash
```

### 2. Frontend

```bash
cd frontend
npm install
npm run dev
```

### 3. Worker build

```bash
cd worker
npm install
npm run build
```

## Environment variables

Backend reads these environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | backend HTTP port |
| `EMAILDASH_DATA_DIR` | `../data` | runtime data directory |
| `EMAILDASH_DB_PATH` | `<data>/emaildash.db` | SQLite path |
| `EMAILDASH_ATTACHMENT_DIR` | `<data>/attachments` | attachment storage path |
| `EMAILDASH_MASTER_KEY_PATH` | `<data>/.masterkey` | AES key file for secrets |
| `EMAILDASH_PUBLIC_BASE_URL` | `http://localhost:8080` | public backend URL used by Worker webhook |
| `EMAILDASH_COOKIE_NAME` | `emaildash_session` | auth cookie name |
| `EMAILDASH_CSRF_HEADER` | `X-CSRF-Token` | CSRF header name |
| `EMAILDASH_WORKER_SCRIPT_NAME` | `emaildash-ingest` | deployed Worker script name |
| `EMAILDASH_WORKER_SUBDOMAIN` | `emaildash-receiver` | Cloudflare Workers subdomain |
| `EMAILDASH_FRONTEND_DIST` | `../frontend/dist` | built frontend path |
| `EMAILDASH_WORKER_BUNDLE` | `../worker/dist/index.js` | built worker bundle path |
| `EMAILDASH_SESSION_TTL_HOURS` | `336` | session lifetime |
| `EMAILDASH_ALLOWED_ORIGIN` | `http://localhost:5173` | frontend dev origin |

## Cloudflare setup notes

Real Cloudflare onboarding needs all of these true:
- selected zone is managed by Cloudflare
- Email Routing can be enabled for zone
- Worker script upload API accepts current payload format
- Worker secret upload succeeds
- catch-all rule creation matches current Cloudflare Email Routing API behavior
- backend webhook is publicly reachable over HTTPS

Recommended staging flow:
1. Build backend, frontend, worker
2. Expose backend publicly with tunnel or deploy host
3. Set `EMAILDASH_PUBLIC_BASE_URL` to that public HTTPS URL
4. Log in to dashboard
5. Save Cloudflare email + Global API Key
6. Choose zone and run provision action
7. Send test mail to any address on selected domain
8. Confirm message lands in inbox

## Security model

- password hashed with Argon2id
- Cloudflare credentials encrypted at rest with local AES-GCM master key
- Worker webhook signed with HMAC SHA-256
- session token stored hashed in DB
- authenticated write endpoints protected by CSRF token

## Important files

### Backend
- `backend/cmd/emaildash/main.go`
- `backend/internal/adapters/http/router.go`
- `backend/internal/adapters/sqlite/db.go`
- `backend/internal/usecase/cloudflare/service.go`
- `backend/internal/usecase/ingest/service.go`
- `backend/internal/platform/crypto/password.go`
- `backend/internal/platform/crypto/secrets.go`
- `backend/internal/platform/signing/hmac.go`

### Frontend
- `frontend/src/app/router.tsx`
- `frontend/src/lib/api.ts`
- `frontend/src/pages/SetupWizard.tsx`
- `frontend/src/pages/Inbox.tsx`
- `frontend/src/pages/SettingsCloudflare.tsx`
- `frontend/src/pages/SettingsPassword.tsx`

### Worker
- `worker/src/index.ts`

## Verify build

### Backend

```bash
cd backend
go build ./...
go test ./...
```

### Frontend

```bash
cd frontend
npm run build
```

### Worker

```bash
cd worker
npm run build
```

### Docker Compose config

```bash
docker compose -f deploy/docker-compose.yml config
```

## Known limitations

- no seed/demo data yet
- no real frontend session bootstrap polish yet after first initialization
- no pagination on frontend inbox yet
- no raw MIME storage yet
- no attachment download endpoint yet
- Cloudflare adapter may need request-shape updates after first live staging test
- `backend/internal/adapters/http/static/index.html` is fallback placeholder, not final embedded app asset pipeline

## Next recommended work

1. finish setup wizard so first-run path initializes password, logs user in, saves Cloudflare credentials, and provisions zone in one pass
2. validate Cloudflare adapter against live staging domain
3. add attachment binary decode + download endpoint
4. embed real frontend build into backend release image
5. add automated tests for auth, ingest, and Cloudflare orchestration

## License

Not added yet.
