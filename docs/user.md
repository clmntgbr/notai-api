# User & authentication

## Overview

Users are provisioned from **Clerk** via webhooks. The API authenticates requests with a **Clerk JWT** (Bearer token).

Each user has a `CurrentClientID` (active tenant). Creating a user also creates a personal client and sets it as current. Switch only via `PUT /api/users/me/current-client`. Non-membership is answered as `404` (same as unknown id) to avoid leaking client existence. Scoped resources (campaigns) use `httpctx.GetCurrentClientID()`.

## Authentication

| Route group | Middleware |
|-------------|------------|
| `/api/*` (protected group) | `AuthenticateMiddleware` — validates JWT, loads/creates user |
| `/webhooks/clerk` | Svix signature verification |

## HTTP routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/users/me` | Current user profile |
| `PUT` | `/api/users/me/current-client` | Switch current client (`{ "clientId" }`) |
| `GET` | `/api/realtime/connection` | Centrifugo connection credentials |

## Code map

| Layer | Location |
|-------|----------|
| HTTP | `internal/interfaces/http/handler/user_handler.go` |
| Commands | `internal/application/command/user/` |
| Queries | `internal/application/query/user/` |
| Domain | `internal/domain/user/` |
| Clerk adapter | `internal/infrastructure/clerk/` |
| Webhook | [Webhooks](webhooks.md) |

## Events

- `user.created.v1`, `user.updated.v1`, `user.deleted.v1` (Clerk webhooks → commands → outbox)
- `user.current_client_changed.v1`

## Tests

`internal/interfaces/http/handler/test/user/`
