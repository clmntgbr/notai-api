-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients (id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    object_key TEXT NOT NULL,
    thumbnail_key TEXT NULL,
    size_bytes BIGINT NULL,
    status TEXT NOT NULL DEFAULT 'pending_upload',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contents_status_check CHECK (
        status IN (
            'pending_upload',
            'uploaded',
            'analyzing',
            'verified',
            'flagged',
            'failed'
        )
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contents_object_key ON contents (object_key);
CREATE INDEX IF NOT EXISTS idx_contents_campaign_id ON contents (campaign_id);
CREATE INDEX IF NOT EXISTS idx_contents_client_id ON contents (client_id);
CREATE INDEX IF NOT EXISTS idx_contents_campaign_status ON contents (campaign_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS contents;
-- +goose StatementEnd
