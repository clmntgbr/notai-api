-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS background_size;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS background_size BIGINT NOT NULL DEFAULT 0;
-- +goose StatementEnd
