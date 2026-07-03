# Nanojira — Task Management API

Task management backend for operations teams, built as a take-home assessment. Inspired by a simplified Jira: managers create and assign work; workers execute and update status; assignments trigger real email notifications.

## Stack

- **Go 1.25** + **Gin** (HTTP)
- **PostgreSQL** (persistence)
- **Mailpit** (local SMTP for assignment emails)
- **Zap** (structured logging)
- **mockgen** (mocks for unit tests)

## Architecture

```
cmd/api/          → entrypoint, dependency wiring
internal/
  domain/         → entities, workflow rules, domain errors
  handler/        → HTTP layer (Gin)
  service/        → use cases (one file per operation)
  repository/     → interfaces + Postgres implementation
  email/          → SMTP delivery
  config/         → environment variables
migrations/       → versioned schema with [goose](https://github.com/pressly/goose) (one migration per table)
mocks/            → generated via mockgen
```

Patterns applied: interfaces, dependency injection, handler → service → repository separation, wrapped errors (`fmt.Errorf("context: %w", err)`), API responses with `code` + `message` so clients can explain failures to users.

## Run with Docker (recommended)

```bash
docker compose up --build
```

Services:

| Service    | URL                        |
|------------|----------------------------|
| API        | http://localhost:8080      |
| Mailpit UI | http://localhost:8025      |
| Postgres   | localhost:5432             |

Health check:

```bash
curl http://localhost:8080/health
```

## Run locally (API without Docker)

```bash
# Start dependencies
docker compose up -d postgres mailpit

# Environment variables (or copy .env.example)
export DATABASE_URL=postgres://nanojira:nanojira@localhost:5432/nanojira?sslmode=disable
export SMTP_HOST=localhost
export SMTP_PORT=1025

make run
# or: go run ./cmd/api
```

Migrations run automatically on API startup via goose. To run them manually:

```bash
make migrate-up       # apply pending migrations
make migrate-down     # revert last migration
make migrate-status   # list applied versions
```

### Migrations (goose)

Each file in `migrations/` covers an isolated context:

| File | Content |
|------|---------|
| `00001_enable_pgcrypto.sql` | Extension for UUIDs |
| `00002_create_users.sql` | `users` table |
| `00003_create_tasks.sql` | `tasks` table + indexes |
| `00004_create_assignment_notifications.sql` | `assignment_notifications` table |
| `00005_create_stepback_requests.sql` | `stepback_requests` table |
| `00006_seed_users.sql` | Initial user data |
| `00007_seed_tasks.sql` | Initial task data |

If the database already existed with the old schema (single file), recreate the volume before starting:

```bash
docker compose down -v && docker compose up --build
```

## Authentication (simulated)

Accounts come from an external system. For testing, use the header:

```
X-User-ID: <user-uuid>
```

### Seed users

| ID | Name | Email | Role |
|----|------|-------|------|
| `11111111-1111-1111-1111-111111111101` | Alice Manager | alice.manager@example.com | manager |
| `11111111-1111-1111-1111-111111111102` | Bob Manager | bob.manager@example.com | manager |
| `22222222-2222-2222-2222-222222222201` | Carol Worker | carol.worker@example.com | worker |
| `22222222-2222-2222-2222-222222222202` | Dave Worker | dave.worker@example.com | worker |
| `22222222-2222-2222-2222-222222222203` | Eve Worker | eve.worker@example.com | worker |

## API

Base path: `/api/v1` — all routes require `X-User-ID`.

| Method | Route | Who | Description |
|--------|-------|-----|-------------|
| GET | `/tasks` | everyone | List tasks (manager: all; worker: assigned only). Query: `status`, `limit`, `offset` |
| POST | `/tasks` | manager | Create task (with or without `assignee_id`) |
| GET | `/tasks/:id` | everyone | Task details |
| PATCH | `/tasks/:id/assign` | manager | Assign worker → sends email |
| PATCH | `/tasks/:id/status` | worker / manager | Worker: advance status or request step-back. Manager: approve/reject pending change |
| GET | `/tasks/:id/notifications` | everyone | Assignment notification history (traceability) |

### Status workflow

```
todo → doing → testing → done
         ↕        ↕
       on_hold ←──┘
```

- **Forward** (worker): `PATCH /tasks/:id/status` with `{"status":"..."}` — applied immediately.
- **Backward** (worker): same endpoint with `{"status":"...", "reason":"..."}` — the task **stays in its current status** and exposes `pending_status_change` until the manager decides.
- **Approval** (manager): `PATCH /tasks/:id/status` with `{"approve_status_change": true}` — moves to the requested status; `false` rejects and keeps the current status.
- Managers **do not** change status directly; they only approve or reject pending changes.

Task response with a pending change:

```json
{
  "id": "...",
  "status": "testing",
  "pending_status_change": {
    "id": "...",
    "requested_status": "doing",
    "reason": "Integration tests failed",
    "requested_by_id": "...",
    "requested_at": "..."
  }
}
```

### Manual test scenario

```bash
# 1. Manager creates task without assignee
curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"title":"Review runbook","description":"Update incident response procedure"}'

# 2. Manager assigns to Carol (email at http://localhost:8025)
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/assign \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"assignee_id":"22222222-2222-2222-2222-222222222201"}'

# 3. Carol advances status
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222201" \
  -d '{"status":"doing"}'

# 4. Dave requests step-back (testing → doing) — status remains "testing"
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 22222222-2222-2222-2222-222222222202" \
  -d '{"status":"doing","reason":"Integration tests failed"}'

# 5. Manager approves the pending change
curl -s -X PATCH http://localhost:8080/api/v1/tasks/<TASK_ID>/status \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 11111111-1111-1111-1111-111111111101" \
  -d '{"approve_status_change":true}'
```

## Tests

```bash
make test          # go test ./...
make mocks         # go generate (mockgen)
```

Unit tests cover: authorization, status transitions, pending step-back, manager approval, and assignment with email.

## Makefile

| Command | Action |
|---------|--------|
| `make run` | Run API locally |
| `make docker-up` | Full stack |
| `make test` | Run tests |
| `make mocks` | Regenerate mocks |
| `make migrate-up` | Apply migrations (goose) |
| `make migrate-down` | Revert last migration |
| `make migrate-status` | Migration status |

---

## Reflections (assessment prompts)

### 1. Scenario interpretation and assumptions

- The system is the **source of truth** for work items: who created them, who is responsible, and what stage they are in.
- **Two roles**: manager (global view, creation, assignment) and worker (own queue, progress).
- Authentication is external → simulated via `X-User-ID`.
- “No step-back without a good reason” → step-back via `PATCH status` with a reason; the task stays at the current step with `pending_status_change` until manager approval (persisted in `stepback_requests`).
- Managers do not execute others’ work → `PATCH status` blocked for managers.
- Assignment email is **real** (SMTP); locally via Mailpit. Each send is recorded in `assignment_notifications` for traceability.

### 2. Important design decisions

- **Explicit workflow** in the domain (`ForwardTransitions` / `BackwardTransitions`) — testable, centralized rules.
- **One file per use case** in the `service/` layer — easier navigation and focused tests.
- **Typed errors** (`AppError` with `code`) — clients can map to UI messages.
- **Repository interfaces** — swappable Postgres; mocks for tests.
- **Pagination** on listings (`limit`/`offset`) — ready for growing volume.

### 3. Assignments and notifications

- Assignment on creation (`assignee_id`) or via `PATCH .../assign`.
- When the assignee changes, SMTP is triggered to the worker’s email.
- Email failure **does not** silently persist the assignment — returns `502 EMAIL_FAILED` (operation fails clearly).
- Record in `assignment_notifications` only after successful delivery.

### 4. Evolution with larger teams/load

- Async queue for emails (SQS/RabbitMQ) and dedicated workers.
- Read cache (Redis) for frequent listings.
- Composite indexes based on query patterns (e.g. `assignee_id + status`).
- Domain events (`task.assigned`, `task.status_changed`) for integrations.
- Real auth (JWT/OAuth2) instead of the header.
- Versioned migrations with goose on boot; in production they would run in a job separate from the API.

### 5. Timebox trade-offs

- Migrations applied on startup via **goose** (versioned, with per-file rollback).
- No integration tests with real Postgres (unit tests with mocks only).
- No separate endpoint to list all manager pending changes (visible via `GET /tasks` with `pending_status_change`).
- No title/description editing after creation.
- Minimal auth (header) — sufficient for the exercise.
---

## AI usage

Tools: **Cursor** (scaffolding, boilerplate, tests, README).

- AI generated the initial folder structure, handlers, and migration SQL.
- I manually reviewed: workflow rules, authorization, error handling, email flow, and tests.
- Without AI, I would have spent more time on Gin/Postgres boilerplate and documentation; the domain design (step-back integrated into status, roles, traceability) was intentional.
