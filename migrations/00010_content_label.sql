-- +goose Up
-- +goose StatementBegin
ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_status_check;
ALTER TABLE contents
    ADD CONSTRAINT contents_status_check CHECK (
        status IN (
            'pending_upload',
            'uploaded',
            'analyzing',
            'failed'
        )
    );

ALTER TABLE contents ADD COLUMN IF NOT EXISTS label TEXT NULL;
ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_label_check;
ALTER TABLE contents
    ADD CONSTRAINT contents_label_check CHECK (
        label IS NULL OR label IN ('human', 'ai_generated', 'uncertain')
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_label_check;
ALTER TABLE contents DROP COLUMN IF EXISTS label;

ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_status_check;
ALTER TABLE contents
    ADD CONSTRAINT contents_status_check CHECK (
        status IN (
            'pending_upload',
            'uploaded',
            'analyzing',
            'verified',
            'flagged',
            'failed'
        )
    );
-- +goose StatementEnd
