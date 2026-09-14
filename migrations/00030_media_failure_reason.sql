-- +goose Up
-- +goose StatementBegin
ALTER TABLE media ADD COLUMN IF NOT EXISTS failure_reason TEXT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE media DROP COLUMN IF EXISTS failure_reason;
-- +goose StatementEnd
