-- +goose Up
-- +goose StatementBegin

-- Idempotent follow-up when 00032 already ran without grace / Pro webhooks.
ALTER TABLE quotas
    ADD COLUMN IF NOT EXISTS quota_overage_grace_verifications INT NOT NULL DEFAULT 0;

UPDATE quotas SET
    quota_overage_grace_verifications = 0,
    updated_at = NOW()
WHERE id = '99f2f76e-dc5c-4131-abf6-0e5e307945fb';

UPDATE quotas SET
    quota_overage_grace_verifications = 5,
    updated_at = NOW()
WHERE id = '1a4a542b-d9ce-40f3-a534-db8e2d24b039';

UPDATE quotas SET
    allows_webhooks = TRUE,
    quota_overage_grace_verifications = 10,
    updated_at = NOW()
WHERE id = '4362cd30-4f10-4c51-9178-e1abe48fce9c';

UPDATE quotas SET
    allows_webhooks = TRUE,
    quota_overage_grace_verifications = 20,
    updated_at = NOW()
WHERE id = 'cfbafc19-0402-4300-8bff-0b6876b362ee';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE quotas
    DROP COLUMN IF EXISTS quota_overage_grace_verifications;

UPDATE quotas SET
    allows_webhooks = FALSE,
    updated_at = NOW()
WHERE id = '4362cd30-4f10-4c51-9178-e1abe48fce9c';

-- +goose StatementEnd
