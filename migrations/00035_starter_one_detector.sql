-- +goose Up
-- +goose StatementBegin
UPDATE quotas SET
    max_detectors_per_analysis = 1,
    updated_at = NOW()
WHERE id = '1a4a542b-d9ce-40f3-a534-db8e2d24b039';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE quotas SET
    max_detectors_per_analysis = 0,
    updated_at = NOW()
WHERE id = '1a4a542b-d9ce-40f3-a534-db8e2d24b039';
-- +goose StatementEnd
