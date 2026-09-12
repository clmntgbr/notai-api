-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients (id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    media_type TEXT NOT NULL,
    object_key TEXT NOT NULL,
    size_bytes BIGINT NULL,
    status TEXT NOT NULL DEFAULT 'pending_upload',
    verdict JSONB NULL,
    analyzed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_media_type_check CHECK (media_type IN ('image', 'video')),
    CONSTRAINT media_status_check CHECK (
        status IN (
            'pending_upload',
            'uploaded',
            'processing',
            'analyzed',
            'failed'
        )
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_media_object_key ON media (object_key);
CREATE INDEX IF NOT EXISTS idx_media_campaign_id ON media (campaign_id);
CREATE INDEX IF NOT EXISTS idx_media_client_id ON media (client_id);
CREATE INDEX IF NOT EXISTS idx_media_campaign_status ON media (campaign_id, status);

-- Backfill one Media per existing Content (image).
INSERT INTO media (
    id,
    campaign_id,
    client_id,
    filename,
    content_type,
    media_type,
    object_key,
    size_bytes,
    status,
    verdict,
    analyzed_at,
    created_at,
    updated_at
)
SELECT
    c.id,
    c.campaign_id,
    c.client_id,
    c.filename,
    c.content_type,
    'image',
    c.object_key,
    c.size_bytes,
    CASE
        WHEN c.status = 'analyzing' THEN 'processing'
        WHEN c.status = 'analyzed' THEN 'analyzed'
        WHEN c.status = 'failed' THEN 'failed'
        WHEN c.status = 'uploaded' THEN 'uploaded'
        ELSE 'pending_upload'
    END,
    CASE
        WHEN c.label IS NOT NULL THEN jsonb_build_object(
            'label', c.label,
            'flaggedCount', CASE WHEN c.label = 'ai_generated' THEN 1 ELSE 0 END,
            'totalCount', 1,
            'failedCount', CASE WHEN c.status = 'failed' THEN 1 ELSE 0 END
        )
        ELSE NULL
    END,
    CASE WHEN c.status = 'analyzed' THEN c.updated_at ELSE NULL END,
    c.created_at,
    c.updated_at
FROM contents c
WHERE NOT EXISTS (SELECT 1 FROM media m WHERE m.id = c.id);

ALTER TABLE contents ADD COLUMN IF NOT EXISTS media_id UUID;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS frame_index INT;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS timestamp_ms BIGINT;

UPDATE contents c
SET media_id = c.id
WHERE media_id IS NULL;

ALTER TABLE contents
    ALTER COLUMN media_id SET NOT NULL;

ALTER TABLE contents
    DROP CONSTRAINT IF EXISTS contents_media_id_fkey;

ALTER TABLE contents
    ADD CONSTRAINT contents_media_id_fkey
    FOREIGN KEY (media_id) REFERENCES media (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_contents_media_id ON contents (media_id);

-- Replace analyzable object keys: keep existing keys; ownership moves to media.
ALTER TABLE contents DROP COLUMN IF EXISTS campaign_id;
ALTER TABLE contents DROP COLUMN IF EXISTS client_id;
ALTER TABLE contents DROP COLUMN IF EXISTS filename;
ALTER TABLE contents DROP COLUMN IF EXISTS content_type;

DROP INDEX IF EXISTS idx_contents_campaign_id;
DROP INDEX IF EXISTS idx_contents_client_id;
DROP INDEX IF EXISTS idx_contents_campaign_status;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE contents ADD COLUMN IF NOT EXISTS campaign_id UUID;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS client_id UUID;
ALTER TABLE contents ADD COLUMN IF NOT EXISTS filename TEXT NOT NULL DEFAULT '';
ALTER TABLE contents ADD COLUMN IF NOT EXISTS content_type TEXT NOT NULL DEFAULT '';

UPDATE contents c
SET
    campaign_id = m.campaign_id,
    client_id = m.client_id,
    filename = m.filename,
    content_type = m.content_type
FROM media m
WHERE c.media_id = m.id;

ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_media_id_fkey;
ALTER TABLE contents DROP COLUMN IF EXISTS media_id;
ALTER TABLE contents DROP COLUMN IF EXISTS frame_index;
ALTER TABLE contents DROP COLUMN IF EXISTS timestamp_ms;

DROP TABLE IF EXISTS media;

-- +goose StatementEnd
