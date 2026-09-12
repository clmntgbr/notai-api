-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS content_analysis_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id UUID NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    detector_name TEXT NOT NULL,
    status TEXT NOT NULL,
    signals JSONB NULL,
    error TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT content_analysis_results_status_check CHECK (
        status IN ('success', 'failed', 'timeout')
    ),
    CONSTRAINT content_analysis_results_content_detector_unique UNIQUE (content_id, detector_name)
);

CREATE INDEX IF NOT EXISTS idx_content_analysis_results_content_id
    ON content_analysis_results (content_id);

ALTER TABLE contents
    ADD COLUMN IF NOT EXISTS confidence DOUBLE PRECISION NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE contents DROP COLUMN IF EXISTS confidence;
DROP TABLE IF EXISTS content_analysis_results;
-- +goose StatementEnd
