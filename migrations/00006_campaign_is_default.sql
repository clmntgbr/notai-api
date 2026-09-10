-- +goose Up
-- +goose StatementBegin
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS is_default BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS idx_campaigns_client_default
    ON campaigns (client_id)
    WHERE is_default = true;

INSERT INTO campaigns (id, client_id, name, is_default, created_at, updated_at, background_status)
SELECT gen_random_uuid(), c.id, 'Default', true, NOW(), NOW(), 'none'
FROM clients c
WHERE NOT EXISTS (
    SELECT 1 FROM campaigns cam
    WHERE cam.client_id = c.id AND cam.is_default = true
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM campaigns WHERE is_default = true;
DROP INDEX IF EXISTS idx_campaigns_client_default;
ALTER TABLE campaigns DROP COLUMN IF EXISTS is_default;
-- +goose StatementEnd
