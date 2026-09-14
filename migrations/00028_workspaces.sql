-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    subscription_id UUID NULL REFERENCES subscriptions (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspaces_owner_user_id UNIQUE (owner_user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspaces_subscription_id ON workspaces (subscription_id);

-- Abort if any owner would lose a second paid subscription when collapsing to one workspace.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM (
            SELECT
                COALESCE(
                    (
                        SELECT uc.user_id
                        FROM user_clients uc
                        WHERE uc.client_id = c.id
                        ORDER BY uc.created_at ASC, uc.user_id ASC
                        LIMIT 1
                    ),
                    (
                        SELECT u.id
                        FROM users u
                        WHERE u.current_client_id = c.id
                        ORDER BY u.created_at ASC
                        LIMIT 1
                    )
                ) AS owner_user_id,
                c.subscription_id
            FROM clients c
            WHERE c.subscription_id IS NOT NULL
        ) owned
        WHERE owner_user_id IS NOT NULL
        GROUP BY owner_user_id
        HAVING COUNT(DISTINCT subscription_id) > 1
    ) THEN
        RAISE EXCEPTION
            'migration 00028: owner has multiple client subscriptions; resolve manually before migrating';
    END IF;
END $$;

-- One workspace per existing client owner; subscription taken from the earliest client.
WITH client_owners AS (
    SELECT
        c.id AS client_id,
        c.subscription_id,
        c.created_at,
        c.updated_at,
        COALESCE(
            (
                SELECT uc.user_id
                FROM user_clients uc
                WHERE uc.client_id = c.id
                ORDER BY uc.created_at ASC, uc.user_id ASC
                LIMIT 1
            ),
            (
                SELECT u.id
                FROM users u
                WHERE u.current_client_id = c.id
                ORDER BY u.created_at ASC
                LIMIT 1
            )
        ) AS owner_user_id
    FROM clients c
),
ranked AS (
    SELECT
        *,
        row_number() OVER (
            PARTITION BY owner_user_id
            ORDER BY created_at ASC, client_id ASC
        ) AS owner_rank
    FROM client_owners
    WHERE owner_user_id IS NOT NULL
)
INSERT INTO workspaces (id, owner_user_id, subscription_id, created_at, updated_at)
SELECT
    gen_random_uuid(),
    owner_user_id,
    subscription_id,
    created_at,
    updated_at
FROM ranked
WHERE owner_rank = 1
ON CONFLICT (owner_user_id) DO NOTHING;

-- Users with no client yet still get an owned workspace (1 workspace / user).
INSERT INTO workspaces (id, owner_user_id, subscription_id, created_at, updated_at)
SELECT
    gen_random_uuid(),
    u.id,
    NULL,
    NOW(),
    NOW()
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM workspaces w WHERE w.owner_user_id = u.id
);

ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS workspace_id UUID;

UPDATE clients c
SET workspace_id = w.id
FROM workspaces w
WHERE c.workspace_id IS NULL
  AND w.owner_user_id = COALESCE(
      (
          SELECT uc.user_id
          FROM user_clients uc
          WHERE uc.client_id = c.id
          ORDER BY uc.created_at ASC, uc.user_id ASC
          LIMIT 1
      ),
      (
          SELECT u.id
          FROM users u
          WHERE u.current_client_id = c.id
          ORDER BY u.created_at ASC
          LIMIT 1
      )
  );

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM clients WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION
            'migration 00028: orphan clients without resolvable owner; fix memberships before migrating';
    END IF;
END $$;

ALTER TABLE clients
    ALTER COLUMN workspace_id SET NOT NULL;

ALTER TABLE clients
    DROP CONSTRAINT IF EXISTS clients_workspace_id_fkey;

ALTER TABLE clients
    ADD CONSTRAINT clients_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_clients_workspace_id ON clients (workspace_id);

ALTER TABLE clients
    DROP COLUMN IF EXISTS subscription_id;

ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS workspace_id UUID;

UPDATE invoices i
SET workspace_id = c.workspace_id
FROM clients c
WHERE i.workspace_id IS NULL
  AND i.client_id = c.id;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM invoices WHERE workspace_id IS NULL) THEN
        RAISE EXCEPTION
            'migration 00028: invoices without remappable client; fix data before migrating';
    END IF;
END $$;

ALTER TABLE invoices
    ALTER COLUMN workspace_id SET NOT NULL;

ALTER TABLE invoices
    DROP CONSTRAINT IF EXISTS invoices_workspace_id_fkey;

ALTER TABLE invoices
    ADD CONSTRAINT invoices_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_invoices_workspace_id ON invoices (workspace_id);

ALTER TABLE invoices
    DROP COLUMN IF EXISTS client_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS client_id UUID REFERENCES clients (id) ON DELETE CASCADE;

-- Prefer the earliest client in the workspace when remapping invoices.
UPDATE invoices i
SET client_id = c.id
FROM (
    SELECT DISTINCT ON (workspace_id) id, workspace_id
    FROM clients
    ORDER BY workspace_id, created_at ASC, id ASC
) c
WHERE i.client_id IS NULL
  AND c.workspace_id = i.workspace_id;

ALTER TABLE invoices
    DROP COLUMN IF EXISTS workspace_id;

ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS subscription_id UUID REFERENCES subscriptions (id) ON DELETE SET NULL;

UPDATE clients c
SET subscription_id = w.subscription_id
FROM workspaces w
WHERE c.workspace_id = w.id;

ALTER TABLE clients
    DROP COLUMN IF EXISTS workspace_id;

DROP TABLE IF EXISTS workspaces;

-- +goose StatementEnd
