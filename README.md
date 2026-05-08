# Emaildash

Self-hosted inbound email dashboard for Cloudflare Email Routing.

Emaildash receives catch-all mail through a Cloudflare Email Worker, verifies a signed webhook, stores messages in SQLite, and serves a single-user dashboard plus REST API from one Go app.

## VPS Install

Prerequisites:
- A VPS with Docker and Docker Compose installed
- A domain or subdomain with an `A` or `AAAA` record pointing to the VPS
- Ports `80` and `443` open
- A Cloudflare-managed domain and Cloudflare Global API Key for email routing setup

Install and start:

```bash
git clone https://github.com/iPurya/emaildash.git
cd emaildash
cp deploy/.env.example .env
DOMAIN=emaildash.example.com
sed -i "s/emaildash.example.com/${DOMAIN}/g" .env
docker compose --env-file .env -f deploy/docker-compose.prod.yml up -d --build
```

Open `https://YOUR_DOMAIN`, create the first password, then use the Cloudflare tab to save credentials and provision the domain.

Runtime data is stored in `./data`:
- `data/emaildash.db`
- `data/.masterkey`
- `data/attachments/`

Back this directory up. Losing `.masterkey` means encrypted Cloudflare credentials and webhook secrets cannot be decrypted.

## Update

```bash
git pull --ff-only
docker compose --env-file .env -f deploy/docker-compose.prod.yml up -d --build
```

## Local Docker

```bash
docker compose -f deploy/docker-compose.yml up --build
```

Open `http://localhost:8080`.

## Stack

- Backend, UI, REST API: Go, Gin, templ, HTMX, Bootstrap
- Database: SQLite via `modernc.org/sqlite`
- Worker: Cloudflare Email Worker, TypeScript, `postal-mime`
- Deploy: Docker Compose and Caddy

## Architecture

```text
Cloudflare Email Routing
  -> catch-all rule
  -> Cloudflare Worker
  -> signed POST /api/ingest/cloudflare/email
  -> Go app
  -> SQLite and attachment storage
  -> dashboard and REST API
```

Main directories:

```text
backend/   Go app, dashboard, REST API, auth, SQLite, Cloudflare automation
worker/    Cloudflare Email Worker source and build config
deploy/    Dockerfile, compose files, Caddyfile, example env
data/      Runtime SQLite DB, master key, and attachments
```

## Important Routes

Browser:
- `GET /`
- `GET /setup`
- `GET /login`
- `GET /dashboard`
- `GET /api/docs`

REST API:
- `GET /api/setup/status`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `GET /api/emails`
- `GET /api/emails/:id`
- `GET /api/recipients`
- `PATCH /api/emails/:id/read`
- `POST /api/cloudflare/credentials`
- `POST /api/cloudflare/zones/:zoneId/provision`
- `POST /api/ingest/cloudflare/email`

Protected REST endpoints accept either the browser session cookie or `?api_key=YOUR_API_KEY`. The dashboard shows the API key under `Password & API`.

## API Examples

```bash
curl "https://emaildash.example.com/api/emails?api_key=YOUR_API_KEY"
curl "https://emaildash.example.com/api/emails?api_key=YOUR_API_KEY&to_mail=test@example.com"
curl "https://emaildash.example.com/api/emails?api_key=YOUR_API_KEY&from_mail=sender@example.com"
curl "https://emaildash.example.com/api/recipients?api_key=YOUR_API_KEY"
```

## Configuration

Production values are read from `.env` by Docker Compose:

```env
EMAILDASH_DOMAIN=emaildash.example.com
EMAILDASH_PUBLIC_BASE_URL=https://emaildash.example.com
EMAILDASH_ALLOWED_ORIGIN=https://emaildash.example.com
```

Backend environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port inside the container |
| `EMAILDASH_DATA_DIR` | `../data` | Runtime data directory |
| `EMAILDASH_DB_PATH` | `<data>/emaildash.db` | SQLite database path |
| `EMAILDASH_ATTACHMENT_DIR` | `<data>/attachments` | Attachment storage path |
| `EMAILDASH_MASTER_KEY_PATH` | `<data>/.masterkey` | AES key file for encrypted secrets |
| `EMAILDASH_PUBLIC_BASE_URL` | `http://localhost:8080` | Public app URL used in Worker webhook config |
| `EMAILDASH_ALLOWED_ORIGIN` | `http://localhost:8080` | CORS origin for API clients |
| `EMAILDASH_WORKER_SCRIPT_NAME` | `emaildash-ingest` | Cloudflare Worker script name |
| `EMAILDASH_WORKER_SUBDOMAIN` | `emaildash-receiver` | Cloudflare Workers subdomain |
| `EMAILDASH_WORKER_BUNDLE` | `../worker/dist/index.js` | Built Worker bundle path |
| `EMAILDASH_SESSION_TTL_HOURS` | `336` | Session lifetime |

## Development

Backend:

```bash
cd backend
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate
go run ./cmd/emaildash
```

Worker:

```bash
cd worker
npm install
npm run build
```

Verification:

```bash
cd backend && go test ./...
cd ../worker && npm run build
docker compose -f deploy/docker-compose.yml config
docker compose --env-file .env -f deploy/docker-compose.prod.yml config
```

## Notes

- Caddy handles HTTPS automatically for `EMAILDASH_DOMAIN`.
- The Cloudflare Worker is built into the app image and uploaded during provisioning.
- Cloudflare automation still depends on live Cloudflare API behavior for the selected account and zone.
- Generated templ files are committed so a fresh clone can build without extra generated artifacts.

## License

MIT
