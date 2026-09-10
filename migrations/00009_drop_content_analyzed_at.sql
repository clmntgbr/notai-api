-- +goose Up
-- +goose StatementBegin
ALTER TABLE contents DROP COLUMN IF EXISTS analyzed_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contents ADD COLUMN IF NOT EXISTS analyzed_at TIMESTAMPTZ NULL;
-- +goose StatementEnd
