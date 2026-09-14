-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_media_status_updated_at ON media (status, updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_media_status_updated_at;
-- +goose StatementEnd
