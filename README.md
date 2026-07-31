# ClickUp Clone — Go + React

## Stack

- **Backend**: Go 1.22, chi router, pgx/v5, gorilla/websocket, JWT
- **Frontend**: React 18, Vite, TanStack Query, Zustand, Tailwind CSS
- **Data**: PostgreSQL 14+, Redis (optional — used for rate limiting)

## Setup

Pick one of the three paths below.

### Option A — Docker Compose (full stack, recommended)

Brings up Postgres, Redis, the Go backend, and the React frontend (served by nginx) with a single command. No host installs of Postgres / Redis / Node required.

Prerequisites:

- Docker 24+ with Compose v2

Steps:

```bash
# 1. (Optional) set a real JWT secret — defaults to change-me-in-production
export JWT_SECRET=$(openssl rand -hex 32)

# 2. Build and start everything
docker compose up --build
```

Open http://localhost:5173 — nginx serves the SPA and proxies `/api` and `/ws` to the backend over the compose network.

Useful commands:

```bash
docker compose logs -f backend       # tail server logs
docker compose down                  # stop (keeps volumes)
docker compose down -v               # stop + wipe Postgres data and uploads
```

Services and ports:
| Service | Image / build | Host port |
|----------|--------------------|-----------|
| frontend | `frontend/` | 5173 |
| backend | `Dockerfile` | 8080 |
| postgres | `postgres:16` | 5432 |
| redis | `redis:7` | 6379 |

Migrations apply automatically on backend startup. Attachments persist in the `backend-storage` volume; Postgres data persists in `postgres-data`.

### Option B — Docker (backend image only)

Builds just the Go server. Use this when you already have Postgres/Redis running elsewhere and don't need the frontend container.

```bash
cp .env.example .env                 # point DB_DSN at your Postgres
docker build -t clickup .
docker run --rm -p 8080:8080 \
  --env-file .env \
  -v clickup-storage:/app/storage \
  clickup
```

On Windows/macOS, use `host.docker.internal` instead of `localhost` in `DB_DSN` / `REDIS_ADDR` so the container can reach services on the host.

### Option C — Local dev (no Docker)

Prerequisites (install locally):

- Go 1.22+
- Node.js 20+
- PostgreSQL 14+ running on `localhost:5432`
- Redis (optional) running on `localhost:6379`

#### Install Postgres on Windows

Download the installer from https://www.postgresql.org/download/windows/ and run:

```bash
createdb -U postgres clickup
```

#### Install Redis on Windows (optional)

Use [Memurai](https://www.memurai.com/) or WSL. If Redis isn't running the server still starts — rate limiting is skipped.

#### Quick start

```bash
# 1. Copy env and edit DB_DSN
cp .env.example .env

# 2. Install Go deps
go mod tidy

# 3. Run the server (auto-applies migrations on boot)
make run

# 4. In another terminal, start the frontend
make fe
```

App runs at http://localhost:5173 — Vite proxies `/api` and `/ws` to `:8080`.

#### Running the Go backend without `make`

On Windows (or anywhere `make` isn't installed), run the equivalent `go` commands directly from the repo root:

```bash
# Apply migrations and exit
go run ./cmd/server -migrate

# Run the server (auto-applies migrations on boot when AUTO_MIGRATE=true)
go run ./cmd/server

.\server.exe

# Build a static binary into ./bin
go build -o bin/server ./cmd/server
./bin/server                          # or .\bin\server.exe on Windows

# Seed the database with sample data
go run ./scripts/seed.go

# Tidy modules
go mod tidy
```

The server reads config from `.env` in the working directory, so always invoke these from the repo root. It listens on `:8080` by default — override with `PORT=...` in `.env`. Logs stream to stdout; press `Ctrl+C` to stop.

## Migrations

The server has a built-in forward-only migration runner — no `golang-migrate` CLI required.
It reads `migrations/*.up.sql` in lexicographic order and tracks applied versions in the `schema_migrations` table.

- Apply automatically on boot: `AUTO_MIGRATE=true` (default).
- Apply and exit: `make migrate`.

## Bootstrap

Self-registration is disabled — there is no public `/auth/register` endpoint.
The first owner is created by the seeder; every subsequent user is invited
by an existing workspace admin.

```bash
go run ./cmd/server -migrate     # apply schema
go run ./scripts/seed.go         # creates demo@demo.test (owner) and three other users
```

Default password for every seeded user: `demo1234` (override with
`SEED_PASSWORD=...` before running). Change it in Settings on first login.

To onboard a real user once an admin exists, the admin POSTs to
`/api/v1/workspaces/{workspaceID}/members` with `{email, name, password, role}` —
the server hashes the supplied password, creates the local account, and
adds the membership row in one call. Share the password with the new user
out-of-band; they log in via `/auth/login` and can change it from Settings.

## API

All endpoints are under `/api/v1`. Auth is via `Authorization: Bearer <token>`.

Public:

- ~~`POST /auth/register`~~ — **removed**: self-registration disabled. Use the
  workspace member-invite flow below to create new users.
- `POST /auth/login` `{ email, password }`
- `POST /auth/mfa/verify` `{ challenge_token, code }` (only after MFA is enrolled)
- `GET  /auth/oidc/{provider}/login` — start an OIDC sign-in (when configured)

Private (require token):

- `GET  /me`
- `POST /workspaces`, `GET /workspaces`
- `POST /workspaces/{workspaceID}/members` `{ email, role }` — invite an **already-registered** user. Returns `404` with `{reason: "user_not_found"}` when the email isn't on file — create the user via the User Management endpoints below.
- `POST /spaces`, `GET /workspaces/{workspaceID}/spaces`, `PATCH /spaces/{id}`, `DELETE /spaces/{id}`
- `POST /folders`, `GET /spaces/{spaceID}/folders`, `DELETE /folders/{id}`
- `POST /lists`, `GET /spaces/{spaceID}/lists`, `GET /folders/{folderID}/lists`
- `POST /tasks`, `GET /tasks?list_id=...&status=...`, `PATCH /tasks/{id}`, `DELETE /tasks/{id}`
- `POST /comments`, `GET /tasks/{taskID}/comments`
- `GET  /permissions`, `GET /workspaces/{id}/roles`, `GET /workspaces/{id}/roles/effective`

User management (require `user.manage` permission — granted to owner + admin built-ins):

- `GET    /workspaces/{wsID}/users` — list workspace users with role & joined-at
- `POST   /workspaces/{wsID}/users` `{ email, name, password, role }` — create account + add as workspace member
- `PATCH  /workspaces/{wsID}/users/{userID}` `{ name }` — update display name
- `POST   /workspaces/{wsID}/users/{userID}/reset-password` `{ password }` — admin password reset
- `DELETE /workspaces/{wsID}/users/{userID}` — hard-delete the account
- `GET  /workspaces/{id}/audit/export?format=csv|json`
- `GET  /workspaces/{id}/gdpr/export`

WebSocket at `/ws` — send `{"action":"join","room":"list:<uuid>"}` to subscribe to task events for that list.

## Project structure

```
clickup/
├── cmd/server/main.go
├── internal/
│   ├── config/           # typed .env loader
│   ├── db/               # pgx pool + redis client
│   ├── migrate/          # forward-only SQL migration runner
│   ├── middleware/       # auth (JWT) / logger / rate-limit
│   ├── httpx/            # JSON helpers
│   ├── ws/               # room-based broadcast hub
│   ├── domain/           # pure structs + repo interfaces
│   ├── handler/<domain>/
│   ├── service/<domain>/
│   └── repository/<domain>/
├── migrations/
├── frontend/
│   ├── src/
│   ├── vite.config.ts
│   ├── Dockerfile        # nginx-served SPA build
│   └── package.json
├── scripts/seed.go
├── Dockerfile            # backend image
├── docker-compose.yml    # full-stack dev (postgres + redis + backend + frontend)
├── Makefile
└── .env.example
```

## Dependency flow

```
handler → service → repository → PostgreSQL
                 ↘ ws.Hub      → WebSocket clients
```

Each layer depends only on the `domain/` interfaces — never on concrete implementations.
