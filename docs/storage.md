# Object storage (MinIO → AWS S3)

Local/dev uses **MinIO**. The Go adapter (`internal/infrastructure/storage/minio.go`) already speaks the **AWS S3 API** (`aws-sdk-go-v2/service/s3`). Switching to real S3 in production is mostly **config + AWS plumbing**, not a rewrite of upload/analysis code.

## What stays the same

| Piece | Notes |
|-------|--------|
| App code paths | Presign → browser/client `PUT` → object-created event → process upload / frames / analysis |
| Port | `domain/port.Storage` (`Put` / `Get` / `Delete` / `PresignedPutURL` + thumbnail variants) |
| Object key layout | Unchanged (`clients/.../campaigns/.../media/...`) |
| DTO for create events | `dto.ObjectCreatedEvent` is already S3 notification–shaped (`Records[].s3.bucket/object`) |

No need to fork a second storage implementation for a first AWS cut: set env vars and point notifications at the existing webhook (or an equivalent bridge).

## Env vars

| Variable | Local (MinIO) | AWS S3 (prod) |
|----------|---------------|----------------|
| `STORAGE_ENDPOINT` | `http://localhost:9000` (host) / public URL used in **presigned** URLs | Leave **empty** (SDK default regional endpoint), or set a custom endpoint / CloudFront origin only if you know why |
| `STORAGE_INTERNAL_ENDPOINT` | `http://minio:9000` (Docker network for server-side Get/Put) | Leave **empty** (same as public/regional) unless you use VPC endpoints |
| `STORAGE_REGION` | `us-east-1` (dummy ok) | Real bucket region, e.g. `eu-west-3` |
| `STORAGE_ACCESS_KEY` | MinIO access key | IAM user access key **or** prefer instance/task role (see below) |
| `STORAGE_SECRET_KEY` | MinIO secret | Matching secret (empty if using role-only credentials later) |
| `STORAGE_BUCKET` | `media` | Prod media bucket name |
| `STORAGE_THUMBNAIL_BUCKET` | `thumbnails` | Prod thumbnails bucket (can be same account, separate bucket) |
| `STORAGE_USE_PATH_STYLE` | `true` (required for MinIO) | **`false`** (virtual-hosted–style: `https://bucket.s3.region.amazonaws.com`) |
| `MINIO_WEBHOOK_SECRET` | Shared bearer for `POST /webhooks/minio/object-created` | Keep a strong secret if you still hit that route; or replace auth when you add an SNS/Lambda bridge |

Example prod-oriented `.env` fragment:

```env
STORAGE_ENDPOINT=
STORAGE_INTERNAL_ENDPOINT=
STORAGE_REGION=eu-west-3
STORAGE_ACCESS_KEY=AKIA...
STORAGE_SECRET_KEY=...
STORAGE_BUCKET=notai-prod-media
STORAGE_THUMBNAIL_BUCKET=notai-prod-thumbnails
STORAGE_USE_PATH_STYLE=false
MINIO_WEBHOOK_SECRET=<long-random>
```

With empty endpoints, `newS3Client` only sets region + credentials and uses the standard AWS S3 endpoint. Presigned PUT URLs then point at S3 so browsers upload **directly to AWS**.

## AWS resources to create

1. **Two buckets** (or one bucket + prefix convention — today the code expects **two bucket names**).
2. **Block public access** on both (uploads use presigned PUT; API streams thumbnails after auth).
3. **CORS** on the **media** bucket (browser `PUT` from the app origin), e.g.:

```json
[
  {
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["PUT", "GET", "HEAD"],
    "AllowedOrigins": ["https://app.example.com"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
```

4. **IAM** policy for the API / workers / analysis / frame-extraction tasks, at least:
   - `s3:PutObject`, `s3:GetObject`, `s3:DeleteObject` on both buckets (and `/*` keys)
   - Optionally `s3:ListBucket` if you add tooling later

5. **Object-created notifications** (this is the main behavioural difference from MinIO).

## Upload notification path (critical)

### Local today

```
MinIO PUT → MinIO webhook → POST /webhooks/minio/object-created
  → MediaUploadWebhookMiddleware (Bearer MINIO_WEBHOOK_SECRET)
  → process media / campaign background
```

Compose wires this with `mc event add` + `notify_webhook` (`compose.dev.yaml`).

### Prod on AWS

S3 does **not** call your HTTP endpoint the same way MinIO does. Typical patterns:

| Pattern | Fit |
|---------|-----|
| **S3 → SNS → HTTPS subscription** to `https://api…/webhooks/minio/object-created` | Closest to current handler; payload is still S3 event JSON. Secure the endpoint (shared secret header, or migrate middleware to SNS signature verification). |
| **S3 → SQS → worker/poller** | More “AWS native”; needs a small consumer that maps the message to `ObjectCreatedDispatcher` (new binary or extend worker). |
| **S3 → Lambda → HTTP/internal** | Thin adapter; good for auth/transform. |

**Do not go live without a notification path.** Without it, presign + client PUT succeed but media stays `pending_upload` / never enters processing.

Checklist:

- [ ] Notification filter on `s3:ObjectCreated:*` (or `Put`) for the media bucket
- [ ] Same event shape consumed by `dto.ObjectCreatedEvent` (or map in the bridge)
- [ ] Auth on the ingress (today: `Authorization: Bearer <MINIO_WEBHOOK_SECRET>`)
- [ ] Thumbnail / non-content keys still ignored by the handler (existing logic)

Optional code tidy (not required for first deploy):

- Rename route `/webhooks/minio/object-created` → `/webhooks/s3/object-created` (alias both)
- Rename `NewMinIOStorage` → `NewS3Storage` (cosmetic)
- Support default AWS credential chain (task role / IRSA) when access key env is empty

## Processes that need the same storage env

All of these call `storage.NewMinIOStorage(env)`:

- `cmd/api` — presign, thumbnail download, webhook side effects
- `cmd/analysis` — read objects for detectors
- `cmd/frameextraction` — read video / write frames

Ship the **same** `STORAGE_*` (and notification target reachable by AWS) to every replica.

## Presign vs internal access

| Concern | Local | AWS |
|---------|--------|-----|
| Presigned URL host | `STORAGE_ENDPOINT` (browser-reachable MinIO) | Regional S3 (empty endpoint) |
| Server Get/Put | `STORAGE_INTERNAL_ENDPOINT` (Docker DNS) | Same regional S3 / VPC endpoint |
| Path style | on | off |

If presigned URLs ever need a custom domain (CloudFront + OAC is usually for **GET**, not PUT), treat that as a separate product decision; default prod is direct S3 PUT.

## CORS / frontend

The app already `PUT`s to the URL returned by `POST /api/medias/presign` (and campaign background presign). After switching buckets:

- Update bucket CORS `AllowedOrigins` to the production front origin(s)
- Confirm the browser is not blocked on preflight (`OPTIONS`)

## Security notes

- Prefer **IAM roles** for ECS/EKS/EC2 over long-lived access keys when you can extend `newS3Client` to use the default credential chain.
- Keep buckets private; never set a public-read ACL for media.
- Rotate `MINIO_WEBHOOK_SECRET` (or SNS auth) independently of S3 keys.
- Thumbnail serving stays behind the API (`GET /api/medias/.../thumbnail` with JWT) — do not expose the thumbnail bucket publicly unless you redesign that flow.

## Smoke test after cutover

1. Presign an image → `PUT` to the returned URL → object appears in the media bucket.
2. Notification hits the API → media leaves `pending_upload` / enters processing.
3. Analysis / frame workers can `Get` the object.
4. Thumbnail appears in the thumbnail bucket and loads via the authenticated API.
5. Delete / replace flows still `Delete` / `DeleteThumbnail` as expected.

## Code map

| Layer | Location |
|-------|----------|
| Config | `internal/infrastructure/config/config.go` (`STORAGE_*`, `MINIO_WEBHOOK_SECRET`) |
| Adapter | `internal/infrastructure/storage/minio.go` |
| Port | `internal/domain/port/storage.go` |
| Webhook HTTP | `media_upload_webhook_handler.go` + middleware |
| Route | `POST /webhooks/minio/object-created` in `cmd/api/routes.go` |
| Local wiring | `compose.dev.yaml` (`minio` + `mc` event setup) |
