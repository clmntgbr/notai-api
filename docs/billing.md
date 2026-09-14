# Billing

Billing is **workspace-scoped**: each user owns one `workspaces` row with a `subscription_id`. Clients (brands) belong to a workspace; invoices and plan quotas follow the workspace. Session exposes **current client** only — billing resolves `client.workspace_id` (host pays).

**Access vs billing:** membership (`user_clients`) decides which clients a user can open. Quotas and subscription always follow the **host workspace** of the active client. An invited guest on brand A consumes the owner's plan for A; their personal signup workspace is unused while they work on A. Checkout and billing portal stay **owner-only**. Any member of a client may invite/remove members on that client (seat quota is per client).

Stripe drives checkout, portal, and invoice sync. Free plan is assigned on user signup (workspace + personal client). Creating additional clients reuses the owner's workspace subscription.

## HTTP routes

### Plans (public)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/plans` | No | List active plans with embedded quotas |

### Subscription & quota (JWT + current client)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/subscription` | Current subscription for the active client's workspace |
| `GET` | `/api/quota` | Usage vs limits (members = current client; campaigns/verifications = whole workspace) |
| `POST` | `/api/subscriptions` | Create Stripe checkout session (or in-place price update) — **workspace owner only** |
| `POST` | `/api/subscriptions/preview` | Preview proration |
| `GET` | `/api/subscriptions/portal` | Stripe billing portal URL — **workspace owner only** |

### Invoices

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/invoices` | Paginated invoices for the active client's workspace |

### Webhook

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/webhooks/stripe` | Stripe signature (`STRIPE_WEBHOOK_SECRET`) |

Checkout uses `client_reference_id` = **workspace UUID**.

## Quotas

Enforced inside application commands (not HTTP middleware). Handlers map quota errors with `respondQuotaError` → HTTP `403`.

| Limit | Enforced on |
|-------|-------------|
| `max_client_members` | `POST /api/clients/:id/members` — counted on the **target client** |
| `max_campaigns` | create campaign — counted across **all clients in the workspace** |
| `max_verifications_per_month` | content analyze (before analysis starts) — workspace-wide; upload/presign is not blocked |
| `max_concurrent_analyses` | content analyze start — workspace-wide (soft retry / deferred) |
| `max_file_size_mb` | media process upload |
| `allows_video_analysis` | media presign / process upload |

**Verification unit** = analyzed **Content** (image or video frame), not Media.

If `overage_price_cents > 0`, verification overage is allowed (no Stripe overage billing in v1). Free plan has `0` → hard block.

`GET /api/medias/stats` → `kpis.planIncluded` comes from `max_verifications_per_month`.

## Config

| Env | Purpose |
|-----|---------|
| `STRIPE_SECRET_KEY` | Stripe API |
| `STRIPE_WEBHOOK_SECRET` | Webhook signature |
| `REDIRECT_SUCCESS_URL` | Checkout success |
| `REDIRECT_CANCEL_URL` | Checkout cancel |
| `REDIRECT_PORTAL_URL` | Customer portal return |

Seed Stripe price IDs in `plans.stripe_price_id` before going live (`price_TODO_*` placeholders in migration `00025`).
