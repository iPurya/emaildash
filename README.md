# Emaildash

Simple inbound email dashboard built with Go, Gin, templ, HTMX, SQLite, and Cloudflare Email Routing.

Goal: minimal setup, single-user auth, automatic Cloudflare domain hookup, catch-all email ingest, grouped inbox by recipient, and REST API access to stored messages.

## Stack

- Backend/UI/API: Go + Gin + templ + HTMX + Bootstrap
- Database: SQLite via `modernc.org/sqlite`
- Worker: TypeScript + Cloudflare Email Worker + `postal-mime`
- Deploy: Docker Compose

## Architecture

```text
Cloudflare Email Routing
  -> Catch-all rule
  -> Cloudflare Worker (`worker/src/index.ts`)
  -> Signed POST webhook
  -> Go app (`/api/ingest/cloudflare/email`)
  -> SQLite + attachment storage
  -> Same Go app serves dashboard + REST API
```

### Main directories

```text
backend/   Go server, templ UI, auth, SQLite, Cloudflare automation, ingest API
worker/    Cloudflare Email Worker
deploy/    Dockerfile and compose files
data/      SQLite DB, secrets, attachments at runtime
```

## Features

### Backend and UI
- single-user setup and password auth
- server-rendered dashboard UI with templ
- HTMX inbox refresh and dashboard interactions
- session cookie auth
- encrypted secret storage for Cloudflare credentials and webhook secret
- SQLite schema and migration bootstrap
- inbox list/detail/read endpoints
- grouped recipients endpoint
- Cloudflare credential save, zone listing, provisioning, and status
- signed ingest webhook

### Worker
- parses inbound email with `postal-mime`
- extracts sender, recipient, subject, text, HTML, headers, attachments
- signs webhook payload with HMAC SHA-256

## Routes

### Browser routes
- `GET /`
- `GET /setup`
- `POST /setup`
- `GET /login`
- `POST /login`
- `POST /logout`
- `GET /dashboard`
- `GET /ui/inbox/recipients`
- `GET /ui/inbox/emails`
- `GET /ui/inbox/viewer`
- `POST /dashboard/password`
- `POST /dashboard/cloudflare/credentials`
- `POST /dashboard/cloudflare/provision`

### JSON API
- `GET /api/setup/status`
- `POST /api/setup/initialize`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/settings/password`
- `POST /api/cloudflare/credentials`
- `GET /api/cloudflare/zones`
- `POST /api/cloudflare/zones/:zoneId/provision`
- `GET /api/cloudflare/status`
- `GET /api/emails` (`recipient`, `to_mail`, `from_mail`, `unread`, `limit` optional filters)
- `GET /api/emails/:id`
- `PATCH /api/emails/:id/read`
- `GET /api/recipients`
- `POST /api/ingest/cloudflare/email`

## Local setup

### Prerequisites
- Go 1.24+
- Node.js 22+
- npm 10+
- Docker Desktop + Docker Compose if using container workflow

Cloudflare requirements for real end-to-end test:
- Cloudflare-managed domain
- Cloudflare account email
- Cloudflare Global API Key
- public HTTPS URL for backend webhook

## Local run

### Backend

```bash
cd backend
go mod tidy
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/emaildash
```

### Worker build

```bash
cd worker
npm install
npm run build
```

Open:
- `http://localhost:8080/`

## Docker Compose

From repo root:

```bash
docker compose -f deploy/docker-compose.yml up --build
```

App:
- `http://localhost:8080`

Mounted runtime data:
- `./data/emaildash.db`
- `./data/attachments/`
- `./data/.masterkey`

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
| `EMAILDASH_WORKER_SCRIPT_NAME` | `emaildash-ingest` | deployed Worker script name |
| `EMAILDASH_WORKER_SUBDOMAIN` | `emaildash-receiver` | Cloudflare Workers subdomain |
| `EMAILDASH_WORKER_BUNDLE` | `../worker/dist/index.js` | built worker bundle path |
| `EMAILDASH_SESSION_TTL_HOURS` | `336` | session lifetime |
| `EMAILDASH_ALLOWED_ORIGIN` | `http://localhost:8080` | app origin |

## Build verification

### Backend

```bash
cd backend
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go build ./...
go test ./...
```

### Worker

```bash
cd worker
npm run build
```

### Docker Compose config

```bash
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml build
```

## Known limitations

- attachment binary persistence still needs a follow-up pass if corruption appears in real mail tests
- Cloudflare adapter still needs broader live validation against more real accounts/zones
- generated templ files are committed to keep clone/build flow simple

## License

MIT
