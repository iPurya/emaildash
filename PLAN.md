# Context

Repo is empty, so this is a greenfield build in `C:\Users\Purya\Documents\vibe\emaildash`.

Target outcome: Docker-first, open-source email receiving dashboard with minimal setup, Go backend, SQLite storage, React frontend, and one guided flow to connect a Cloudflare domain, enable Email Routing, deploy a Worker, and ingest inbound mail into dashboard/API.

Chosen stack:
- Backend: Go + Gin + pure-Go SQLite driver (`modernc.org/sqlite`)
- Frontend: React + Vite + Tailwind + shadcn/ui
- Worker: TypeScript module Worker
- Packaging: `docker compose` for dev, multi-stage Docker build for prod

Existing reusable code in repo: none found. Repo is empty, so all implementation is net-new.

## Recommended approach

### 1. Bootstrap repo as three apps plus deploy assets
Create four top-level areas:
- `backend/` for Go server, DB, auth, Cloudflare automation, ingest API
- `frontend/` for React dashboard and setup wizard
- `worker/` for Cloudflare Email Worker source and build
- `deploy/` for Dockerfiles and `docker-compose.yml`

Keep final production image single-container: build frontend + worker artifacts during Docker build, then serve frontend from Go binary and upload embedded worker bundle through backend automation.

### 2. Build backend with clean architecture and narrow adapters
Use this backend layout:
- `backend/cmd/emaildash/main.go`
- `backend/internal/domain/`
- `backend/internal/usecase/`
- `backend/internal/ports/`
- `backend/internal/adapters/http/`
- `backend/internal/adapters/sqlite/`
- `backend/internal/adapters/cloudflare/`
- `backend/internal/platform/`

Keep domain and usecase layers free of HTTP, SQLite, and Cloudflare SDK details. Cloudflare and SQLite stay behind interfaces so setup/auth/ingest logic stays testable.

### 3. Implement SQLite schema for single-user auth, config, and inbox
Create embedded SQL migrations for:
- `app_state` — initialized flag and app metadata
- `users` — single local user password hash and timestamps
- `sessions` — hashed session tokens and expiry
- `secrets` — encrypted Cloudflare email, Global API key, webhook secret
- `cloudflare_zones` — cached zone/account selection and status
- `emails` — normalized inbound message record
- `email_recipients` — one row per recipient address for grouping/filtering
- `attachments` — metadata plus disk path
- `audit_log` — setup and Cloudflare automation events

Store attachments on disk under mounted data volume, not as SQLite BLOBs. Keep only metadata and file path in DB.

### 4. Implement setup and auth flow first
Backend endpoints:
- `GET /api/setup/status`
- `POST /api/setup/initialize`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `POST /api/settings/password`

Rules:
- App starts in uninitialized mode
- Setup wizard creates first password once
- Password hashing uses Argon2id
- Session auth uses secure HttpOnly cookie with hashed server-side token
- Add CSRF protection for authenticated writes
- Rate-limit login and setup endpoints

### 5. Implement Cloudflare automation as one orchestration service
Create `backend/internal/usecase/cloudflare/service.go` to drive this sequence:
1. Accept Cloudflare account email + Global API key from setup/settings UI
2. Verify credentials and list zones/domains
3. Persist credentials encrypted at rest
4. When user selects zone, fetch zone details and account ID
5. Check Email Routing status for zone
6. Enable/onboard Email Routing through current Cloudflare API adapter
7. Upload or update Worker script for that account
8. Push Worker secrets:
   - `EMAILDASH_WEBHOOK_URL`
   - `EMAILDASH_WEBHOOK_SECRET`
9. Bind catch-all email routing rule to Worker action
10. Save selected zone and resulting status for UI

Important implementation note: isolate Cloudflare Email Routing enable/update logic inside adapter because current official docs still expose enable endpoint but mark it deprecated. Keep this detail out of usecase layer so API churn stays localized.

Handle Cloudflare blockers explicitly in API/UI:
- conflicting MX records
- zone not fully onboarded for Email Routing
- Worker subdomain/bootstrap not ready yet
- missing public HTTPS webhook URL

### 6. Build Worker as thin parser + signed forwarder
Worker entrypoint: `worker/src/index.ts`

Responsibilities:
- Use module Worker `email(message, env, ctx)` handler
- Parse `message.raw` with `postal-mime`
- Extract:
  - envelope from/to
  - RFC `Message-ID`
  - subject
  - text body
  - HTML body
  - normalized headers
  - raw size
  - attachment metadata and bytes
- Write attachment bytes into JSON-safe transport format only if small; otherwise send metadata first and keep v1 attachment persistence limited to metadata until backend file-upload path is added
- POST payload to backend webhook
- Sign request with HMAC SHA-256 over `timestamp.body`
- Send headers:
  - `X-Emaildash-Timestamp`
  - `X-Emaildash-Signature`

Keep Worker logic focused on parsing and delivery. No business logic, no dashboard-specific grouping.

### 7. Implement secure ingest webhook and normalized inbox API
Backend ingest endpoint:
- `POST /api/ingest/cloudflare/email`

Ingest behavior:
- verify HMAC signature with constant-time compare
- reject stale timestamps outside 5-minute window
- optional replay cache keyed by signature/timestamp
- normalize sender, recipient list, subject, bodies, headers, attachment metadata
- dedupe on provider + provider message ID when available
- persist email, recipients, and attachment metadata/path

Expose dashboard/API endpoints:
- `GET /api/emails`
- `GET /api/emails/:id`
- `PATCH /api/emails/:id/read`
- `GET /api/recipients`

`GET /api/recipients` should return grouped recipient counts and latest message info so UI can group inbox by destination address without client-side full scan.

### 8. Build React app around setup wizard, grouped inbox, and settings
Create pages:
- `frontend/src/pages/SetupWizard.tsx`
- `frontend/src/pages/Login.tsx`
- `frontend/src/pages/Inbox.tsx`
- `frontend/src/pages/EmailDetail.tsx`
- `frontend/src/pages/SettingsCloudflare.tsx`
- `frontend/src/pages/SettingsPassword.tsx`

Frontend behavior:
- Setup wizard steps: password -> Cloudflare credentials -> zone pick -> automation run -> success
- Inbox layout: recipient groups sidebar, email list, detail pane
- Settings page: rotate password, reconnect or re-run Cloudflare automation
- Data fetching via TanStack Query
- Render HTML email body only after sanitizing with DOMPurify

### 9. Critical files to create

Backend
- `backend/cmd/emaildash/main.go`
- `backend/internal/adapters/http/router.go`
- `backend/internal/adapters/http/handlers/setup.go`
- `backend/internal/adapters/http/handlers/auth.go`
- `backend/internal/adapters/http/handlers/cloudflare.go`
- `backend/internal/adapters/http/handlers/ingest.go`
- `backend/internal/adapters/http/handlers/emails.go`
- `backend/internal/usecase/cloudflare/service.go`
- `backend/internal/usecase/ingest/service.go`
- `backend/internal/adapters/cloudflare/client.go`
- `backend/internal/adapters/sqlite/db.go`
- `backend/internal/adapters/sqlite/migrations/001_init.sql`
- `backend/internal/platform/crypto/password.go`
- `backend/internal/platform/crypto/secrets.go`
- `backend/internal/platform/signing/hmac.go`

Worker
- `worker/src/index.ts`
- `worker/package.json`
- `worker/tsconfig.json`

Frontend
- `frontend/src/app/router.tsx`
- `frontend/src/lib/api.ts`
- `frontend/src/pages/SetupWizard.tsx`
- `frontend/src/pages/Inbox.tsx`
- `frontend/src/components/inbox/RecipientGroups.tsx`
- `frontend/src/components/inbox/EmailList.tsx`
- `frontend/src/components/inbox/EmailViewer.tsx`

Deploy
- `deploy/docker/Dockerfile.backend`
- `deploy/docker/Dockerfile.frontend`
- `deploy/docker-compose.yml`

### 10. Verification

Backend verification:
- run unit tests for password hashing, session auth, HMAC verification, email normalization, and Cloudflare orchestration with mocked adapter
- run integration tests against temp SQLite DB for setup, login, ingest, and email listing

Worker verification:
- run `wrangler dev` local email handler test with sample raw `.eml`
- confirm parsed subject/text/html/headers/attachments match fixtures
- confirm signed POST hits local/mock backend contract correctly

Full app verification:
- `docker compose up` starts backend, frontend, and mounted SQLite volume
- complete setup wizard end-to-end
- simulate signed ingest request and confirm inbox render/grouping by recipient
- verify password change and re-login

Cloudflare staging verification:
- use a real Cloudflare-managed test domain and public HTTPS app URL or tunnel
- connect credentials, select zone, run automation
- confirm Email Routing becomes ready, Worker deploys, catch-all rule points to Worker, and inbound email appears in inbox
- verify grouped recipient view and external REST read endpoint against real stored message
