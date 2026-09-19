-- +goose Up
-- +goose StatementBegin
ALTER TABLE media
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_media_client_id_active
    ON media (client_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_media_campaign_id_active
    ON media (campaign_id)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_media_campaign_id_active;
DROP INDEX IF EXISTS idx_media_client_id_active;
ALTER TABLE media DROP COLUMN IF EXISTS deleted_at;
-- +goose StatementEnd
