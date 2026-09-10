# NotAI API

Go/Fiber backend for NotAI: **Clean Architecture**, **CQRS**, transactional **outbox**, **RabbitMQ**, **Centrifugo**, **PostgreSQL**, and **Clerk** auth.

Multi-tenant around **clients** (membership via `user_clients`) and **campaigns** scoped by the user’s `currentClientId`.

## Features

- CQRS: command / query / event handlers under `internal/application`
- Aggregates: `user`, `client`, `campaign` with versioned domain events + outbox
- User signup (Clerk webhook or JIT on first JWT) creates a personal client and sets `currentClientId` in the same transaction
- RabbitMQ worker (outbox relay + consumer + handler registry) on `domain.events`
- Centrifugo realtime (outbox → worker → `users:{id}`)
- Offset pagination for client / campaign lists
- Clerk JWT auth + webhook sync (Svix)
- Docker Compose local stack (API, worker, Postgres, RabbitMQ, Centrifugo, MinIO, Mailpit, ngrok)

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP | Fiber v3 |
| ORM | GORM + PostgreSQL |
| Auth | Clerk (JWKS + webhooks) |
| Messaging | RabbitMQ (topic exchange `domain.events`) |
| Realtime | Centrifugo v5 |
| CLI | Cobra (goose migrations) |
| Dev | Docker, Air, golangci-lint |

## Architecture

```
cmd/
  api/          HTTP server
  worker/       Outbox relay + RabbitMQ consumer
  cli/          Migrations
internal/
  domain/                 user, client, campaign, event, paginate, port
  application/
    command/              Write-side handlers
    query/                Read-side handlers
    event/                Async handlers (+ realtime publish)
    realtime/             WS type helpers (entity.action)
    registry/             Event handler registry
  infrastructure/
    persistence/write|read|outbox|processed
    messaging/rabbitmq
    centrifugo
    clerk|config
  interfaces/http/        Handlers, middleware, DTO, presenter, validation
migrations/               Goose SQL (schema source of truth)
```

### Tenant model

```
User ──< user_clients >── Client ──< Campaign
  │
  └── current_client_id → Client (active tenant)
```

- A user can belong to several clients; a client has several members.
- Campaigns always use `httpctx.GetCurrentClientID()` — never a `clientId` from the body.
- Switch active client: `PUT /api/users/me/current-client` with `{ "clientId" }`.
- Unknown id and non-member both return **404** (no existence leak). Cross-client campaign access for a member of the other client returns **409** + `WRONG_ORGANIZATION`.

### Sync command + async event flow

```
HTTP / webhook
  → command (same DB transaction)
      → save aggregate(s)
      → StoreEvents(outbox)
  → HTTP response

worker (outbox relay)
  → poll unpublished outbox rows
  → publish to RabbitMQ exchange domain.events
  → mark published_at

worker (consumer)
  → HandlerRegistry.Dispatch(event type)
  → log / notify / publish_*_realtime → Centrifugo users:{userId}
```

### Domain events (outbox)

| Aggregate | Types |
|---|---|
| User | `user.created.v1`, `user.updated.v1`, `user.deleted.v1`, `user.current_client_changed.v1` |
| Client | `client.created.v1`, `client.updated.v1`, `client.deleted.v1`, `client.member_added.v1`, `client.member_removed.v1` |
| Campaign | `campaign.created.v1`, `campaign.updated.v1`, `campaign.deleted.v1` |

### Idempotence + retry / DLQ

- Stable `eventId` when the event is recorded on the aggregate.
- Dedup on `(event_id, handler_name)` via `processed_events`.
- Topology: `domain.events` → `domain.events.retry` (TTL) → main; poison / max attempts → `domain.events.dlq`.
- Never `Nack(requeue=true)`.

If queue declare fails after changing args, delete the old queues (or recreate the RabbitMQ volume) then restart the worker.

## Getting Started

```bash
cp .env.dist .env
# fill Clerk keys (+ NGROK_AUTHTOKEN if you need webhooks)

make dev
make migrate
```

| Service | Host port | Notes |
|---|---|---|
| API | `4000` | → container `3000`, Air hot reload |
| Worker | — | outbox relay + consumer |
| Postgres | `9543` | |
| RabbitMQ | `9002` / UI `9003` | user/password from `.env` |
| Centrifugo | `8000` | WS + HTTP API |
| MinIO | `9000` / console `9001` | |
| Mailpit | SMTP `9025` / UI `9026` | |
| ngrok | `4040` | webhook tunnel |

## Makefile

| Command | Description |
|---|---|
| `make dev` | Start compose.dev stack |
| `make restart` | Restart **api** and **worker** only |
| `make build` | Start stack with `--build` |
| `make dev-down` | Stop stack |
| `make api-logs` | Follow API logs |
| `make worker-logs` | Follow worker logs |
| `make tests` | Handler HTTP tests (in api container) |
| `make coverage` | Handler coverage profile |
| `make coverage-html` | HTML coverage report |
| `make migrate` | Apply SQL migrations + schema drift check |
| `make migrate-check` | Fail if persistence models ≠ DB columns |
| `make migrate-status` | Goose migration status |
| `make migrate-down` | Roll back one migration |
| `make lint` | golangci-lint --fix |
| `make shell` | Shell into API container |

## Testing

Handler HTTP tests live under `internal/interfaces/http/handler/test/` (`user`, `client`, `campaign`, `user_webhook`, `realtime`). See [docs/handler-tests.md](docs/handler-tests.md).

```bash
go test ./internal/interfaces/http/handler/test/... -race \
  -coverpkg=./internal/interfaces/http/handler/...
```

CI enforces ≥95% handler statement coverage (Codecov flag `handler`, patch 100%).

## API

All `/api/*` routes below (except health) require `Authorization: Bearer <Clerk JWT>`.

### Health

- `GET /livez`, `/readyz`, `/startupz`

### User

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/me` | Current user (`currentClientId`, …) |
| `PUT` | `/api/users/me/current-client` | Switch active client — body `{ "clientId" }` |

### Clients

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/clients` | List memberships (paginated) |
| `POST` | `/api/clients` | Create client (adds creator + sets current) |
| `GET` | `/api/clients/:id` | Get by id |
| `PUT` | `/api/clients/:id` | Update |
| `DELETE` | `/api/clients/:id` | Delete |
| `DELETE` | `/api/clients/:id/members/:userId` | Remove member (clears their current if needed) |

### Campaigns

Scoped by the caller’s `currentClientId`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/campaigns` | List for current client (paginated) |
| `POST` | `/api/campaigns` | Create — body `{ "name" }` |
| `GET` | `/api/campaigns/:id` | Get (409 `WRONG_ORGANIZATION` if other client you belong to) |
| `PUT` | `/api/campaigns/:id` | Update |
| `DELETE` | `/api/campaigns/:id` | Delete |

### Realtime

- `GET /api/realtime/connection` — Centrifugo JWT + channel + `wsUrl`

### Webhooks

- `POST /webhooks/clerk` — Svix-verified; `user.created` / `updated` / `deleted`

## Environment

See [`.env.dist`](.env.dist). Messaging / realtime:

- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE` (default `domain.events`)
- `RABBITMQ_QUEUE` (default `domain.events`)
- `RABBITMQ_ROUTING_KEY` (default `user.#,client.#,campaign.#`, comma-separated)
- `OUTBOX_POLL_INTERVAL` (default `2s`)
- `WORKER_CONCURRENCY` (default `4`)
- `CENTRIFUGO_URL`, `CENTRIFUGO_API_KEY`, `CENTRIFUGO_TOKEN_SECRET`, `CENTRIFUGO_PUBLIC_WS_URL`
- `RATE_LIMIT_MAX` (requests per minute per IP; `0` disables)

## Extending

1. Add aggregate under `internal/domain/<name>/` (`entity.go`, `events.go`, `repository.go`)
2. Add command/query handlers under `internal/application/...`
3. Implement write/read repos under `internal/infrastructure/persistence/`
4. Register event handlers in `cmd/worker/di` (including `publish_*_realtime` if needed)
5. Wire HTTP in `cmd/api/di` + `routes.go` (+ handler tests under `handler/test/<resource>/`)
6. Add a SQL migration in `migrations/` and matching persistence models
7. Broaden `RABBITMQ_ROUTING_KEY` if you add a new event prefix
8. Run `make migrate` (SQL + model/DB drift check)

## Docs

- Architecture conventions: [`.cursor/rules/architecture.mdc`](.cursor/rules/architecture.mdc)
- User / auth / current client: [docs/user.md](docs/user.md)
- Webhooks: [docs/webhooks.md](docs/webhooks.md)
- Realtime: [docs/realtime.md](docs/realtime.md)
- Handler tests: [docs/handler-tests.md](docs/handler-tests.md)
