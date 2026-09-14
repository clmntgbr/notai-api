-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans (id),
    stripe_customer_id VARCHAR(255) NULL,
    stripe_subscription_id VARCHAR(255) NULL,
    status VARCHAR(255) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    quota_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id ON subscriptions (plan_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions (status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id
    ON subscriptions (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL AND stripe_subscription_id <> '';
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_customer_id
    ON subscriptions (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL AND stripe_customer_id <> '';

ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS subscription_id UUID REFERENCES subscriptions (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_clients_subscription_id ON clients (subscription_id);

-- One Free subscription per existing client that has none.
WITH missing AS (
    SELECT c.id AS client_id
    FROM clients c
    WHERE c.subscription_id IS NULL
),
created AS (
    INSERT INTO subscriptions (
        id, plan_id, status, start_date, end_date,
        cancel_at_period_end, quota_period_start, created_at, updated_at
    )
    SELECT
        gen_random_uuid(),
        '889ea757-54d8-47ee-b8cd-7fece66a0a04'::uuid,
        'active',
        NOW(),
        NOW() + INTERVAL '100 years',
        FALSE,
        NOW(),
        NOW(),
        NOW()
    FROM missing
    RETURNING id
),
numbered_clients AS (
    SELECT client_id, row_number() OVER (ORDER BY client_id) AS rn FROM missing
),
numbered_subs AS (
    SELECT id AS subscription_id, row_number() OVER (ORDER BY id) AS rn FROM created
)
UPDATE clients c
SET subscription_id = ns.subscription_id
FROM numbered_clients nc
JOIN numbered_subs ns ON ns.rn = nc.rn
WHERE c.id = nc.client_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_clients_subscription_id;
ALTER TABLE clients DROP COLUMN IF EXISTS subscription_id;

DROP INDEX IF EXISTS idx_subscriptions_stripe_customer_id;
DROP INDEX IF EXISTS idx_subscriptions_stripe_subscription_id;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_plan_id;
DROP TABLE IF EXISTS subscriptions;

-- +goose StatementEnd
