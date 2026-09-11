-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS start_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS end_at TIMESTAMPTZ NULL;

ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_schedule_check;
ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_schedule_check CHECK (
        end_at IS NULL OR start_at IS NULL OR end_at >= start_at
    );

CREATE INDEX IF NOT EXISTS idx_campaigns_start_at ON campaigns (start_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_campaigns_start_at;
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_schedule_check;
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS start_at,
    DROP COLUMN IF EXISTS end_at;
-- +goose StatementEnd
