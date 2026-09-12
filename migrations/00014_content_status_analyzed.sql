-- +goose Up
-- +goose StatementBegin
ALTER TABLE contents DROP CONSTRAINT IF EXISTS contents_status_check;
ALTER TABLE contents
    ADD CONSTRAINT contents_status_check CHECK (
        status IN (
            'pending_upload',
            'uploaded',
            'analyzing',
            'analyzed',
            'failed'
        )
    );

-- Contents already labeled under the previous model are complete analyses.
UPDATE contents
SET status = 'analyzed'
WHERE status = 'uploaded'
  AND label IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE contents
SET status = 'uploaded'
WHERE status = 'analyzed';

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
-- +goose StatementEnd
