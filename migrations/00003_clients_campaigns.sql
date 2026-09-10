-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_clients (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, client_id)
);

CREATE INDEX IF NOT EXISTS idx_user_clients_client_id ON user_clients (client_id);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS current_client_id UUID NULL
        REFERENCES clients (id) ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_client_id ON campaigns (client_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS campaigns;
ALTER TABLE users DROP COLUMN IF EXISTS current_client_id;
DROP TABLE IF EXISTS user_clients;
DROP TABLE IF EXISTS clients;
-- +goose StatementEnd
