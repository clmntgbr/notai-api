-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS background_status TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS background_pending_key TEXT NULL,
    ADD COLUMN IF NOT EXISTS background_thumbnail_key TEXT NULL,
    ADD COLUMN IF NOT EXISTS background_filename TEXT NULL,
    ADD COLUMN IF NOT EXISTS background_content_type TEXT NULL;

ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS campaigns_background_status_check;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_background_status_check
    CHECK (background_status IN ('none', 'pending', 'ready', 'failed'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_campaigns_background_pending_key
    ON campaigns (background_pending_key)
    WHERE background_pending_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_campaigns_background_pending_key;
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_background_status_check;
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS background_status,
    DROP COLUMN IF EXISTS background_pending_key,
    DROP COLUMN IF EXISTS background_thumbnail_key,
    DROP COLUMN IF EXISTS background_filename,
    DROP COLUMN IF EXISTS background_content_type;
-- +goose StatementEnd
