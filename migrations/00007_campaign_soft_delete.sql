-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_campaigns_client_id_active
    ON campaigns (client_id)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_campaigns_client_id_active;
ALTER TABLE campaigns DROP COLUMN IF EXISTS deleted_at;
-- +goose StatementEnd
