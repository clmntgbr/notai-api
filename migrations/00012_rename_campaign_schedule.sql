-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'started_at'
    ) THEN
        ALTER TABLE campaigns RENAME COLUMN started_at TO start_at;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'ended_at'
    ) THEN
        ALTER TABLE campaigns RENAME COLUMN ended_at TO end_at;
    END IF;
END $$;

ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_schedule_check;
ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_schedule_check CHECK (
        end_at IS NULL OR start_at IS NULL OR end_at >= start_at
    );

DROP INDEX IF EXISTS idx_campaigns_started_at;
CREATE INDEX IF NOT EXISTS idx_campaigns_start_at ON campaigns (start_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_campaigns_start_at;

ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_schedule_check;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'start_at'
    ) THEN
        ALTER TABLE campaigns RENAME COLUMN start_at TO started_at;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'end_at'
    ) THEN
        ALTER TABLE campaigns RENAME COLUMN end_at TO ended_at;
    END IF;
END $$;

ALTER TABLE campaigns
    ADD CONSTRAINT campaigns_schedule_check CHECK (
        ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at
    );
CREATE INDEX IF NOT EXISTS idx_campaigns_started_at ON campaigns (started_at);
-- +goose StatementEnd
