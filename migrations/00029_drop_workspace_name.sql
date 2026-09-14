-- +goose Up
-- +goose StatementBegin
ALTER TABLE workspaces DROP COLUMN IF EXISTS name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE workspaces ALTER COLUMN name DROP DEFAULT;
-- +goose StatementEnd
