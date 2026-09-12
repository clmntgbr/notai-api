-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS heuristic_rulesets (
    version INT PRIMARY KEY,
    params JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_heuristic_rulesets_one_active
    ON heuristic_rulesets ((is_active))
    WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS known_ai_content_hashes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phash TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    label TEXT NOT NULL DEFAULT 'ai_generated',
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT known_ai_content_hashes_phash_key UNIQUE (phash)
);

CREATE INDEX IF NOT EXISTS idx_known_ai_content_hashes_phash
    ON known_ai_content_hashes (phash);

ALTER TABLE content_analysis_results
    ADD COLUMN IF NOT EXISTS ruleset_version INT NULL;

INSERT INTO heuristic_rulesets (version, params, is_active, created_at, notes)
VALUES (
    1,
    '{
      "shortCircuitOn": ["c2pa_generative", "known_hash_match"],
      "shortCircuitMinWeight": 0.8,
      "checks": {
        "exif": {
          "enabled": true,
          "weights": { "exif_missing": 0.15, "c2pa_generative": 0.95 }
        },
        "jpeg_quant": {
          "enabled": true,
          "weights": { "jpeg_quant_generic": 0.25 }
        },
        "prnu": {
          "enabled": false,
          "thresholds": { "prnu_min_correlation": 0.15 },
          "weights": { "prnu_low_correlation": 0.35 }
        },
        "frequency": {
          "enabled": true,
          "thresholds": { "frequency_peak_min": 0.35, "frequency_periodicity_max": 0.55 },
          "weights": { "frequency_artifact": 0.4 }
        },
        "geometry": {
          "enabled": false,
          "weights": { "anatomy_finger_count": 0.5 }
        },
        "patch_noise": {
          "enabled": true,
          "params": { "patch_grid_size": 8, "patch_outlier_max_count": 6 },
          "thresholds": { "patch_variance_stddev": 2.0 },
          "weights": { "patch_noise_inconsistent": 0.35 }
        },
        "known_hash": {
          "enabled": true,
          "params": { "hash_max_distance": 8 },
          "weights": { "known_hash_match": 0.9 }
        }
      }
    }'::jsonb,
    TRUE,
    NOW(),
    'Initial heuristic ruleset — PRNU/geometry disabled pending external runtime'
)
ON CONFLICT (version) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE content_analysis_results DROP COLUMN IF EXISTS ruleset_version;
DROP TABLE IF EXISTS known_ai_content_hashes;
DROP TABLE IF EXISTS heuristic_rulesets;
-- +goose StatementEnd
