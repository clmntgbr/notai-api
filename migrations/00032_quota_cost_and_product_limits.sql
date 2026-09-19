-- +goose Up
-- +goose StatementBegin

ALTER TABLE quotas
    ADD COLUMN IF NOT EXISTS max_detectors_per_analysis INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_frames_per_video INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS allows_reanalysis BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS max_storage_gb INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS max_batch_upload_size INT NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS frame_retention_days INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS allows_custom_ruleset BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS allows_white_label_report BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS allows_webhooks BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS quota_overage_grace_verifications INT NOT NULL DEFAULT 0;

-- Free: local heuristics only, no video, tight storage/batch.
UPDATE quotas SET
    max_detectors_per_analysis = 0,
    max_frames_per_video = 0,
    allows_reanalysis = FALSE,
    max_storage_gb = 1,
    max_batch_upload_size = 5,
    frame_retention_days = 1,
    allows_custom_ruleset = FALSE,
    allows_white_label_report = FALSE,
    allows_webhooks = FALSE,
    quota_overage_grace_verifications = 0,
    updated_at = NOW()
WHERE id = '99f2f76e-dc5c-4131-abf6-0e5e307945fb';

-- Starter: one external detector, no video; slightly higher storage & batch.
UPDATE quotas SET
    max_detectors_per_analysis = 1,
    max_frames_per_video = 0,
    allows_reanalysis = FALSE,
    max_storage_gb = 5,
    max_batch_upload_size = 10,
    frame_retention_days = 7,
    allows_custom_ruleset = FALSE,
    allows_white_label_report = FALSE,
    allows_webhooks = FALSE,
    quota_overage_grace_verifications = 5,
    updated_at = NOW()
WHERE id = '1a4a542b-d9ce-40f3-a534-db8e2d24b039';

-- Pro: one external detector, 12 frames, reanalysis, white-label, webhooks.
UPDATE quotas SET
    max_detectors_per_analysis = 1,
    max_frames_per_video = 12,
    allows_reanalysis = TRUE,
    max_storage_gb = 50,
    max_batch_upload_size = 20,
    frame_retention_days = 30,
    allows_custom_ruleset = FALSE,
    allows_white_label_report = TRUE,
    allows_webhooks = TRUE,
    quota_overage_grace_verifications = 10,
    updated_at = NOW()
WHERE id = '4362cd30-4f10-4c51-9178-e1abe48fce9c';

-- Business: multi-provider, 30 frames, custom ruleset + webhooks.
UPDATE quotas SET
    max_detectors_per_analysis = 3,
    max_frames_per_video = 30,
    allows_reanalysis = TRUE,
    max_storage_gb = 250,
    max_batch_upload_size = 50,
    frame_retention_days = 90,
    allows_custom_ruleset = TRUE,
    allows_white_label_report = TRUE,
    allows_webhooks = TRUE,
    quota_overage_grace_verifications = 20,
    updated_at = NOW()
WHERE id = 'cfbafc19-0402-4300-8bff-0b6876b362ee';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE quotas
    DROP COLUMN IF EXISTS max_detectors_per_analysis,
    DROP COLUMN IF EXISTS max_frames_per_video,
    DROP COLUMN IF EXISTS allows_reanalysis,
    DROP COLUMN IF EXISTS max_storage_gb,
    DROP COLUMN IF EXISTS max_batch_upload_size,
    DROP COLUMN IF EXISTS frame_retention_days,
    DROP COLUMN IF EXISTS allows_custom_ruleset,
    DROP COLUMN IF EXISTS allows_white_label_report,
    DROP COLUMN IF EXISTS allows_webhooks,
    DROP COLUMN IF EXISTS quota_overage_grace_verifications;

-- +goose StatementEnd
