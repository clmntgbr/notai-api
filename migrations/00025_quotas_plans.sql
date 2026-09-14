-- +goose Up
-- +goose StatementBegin

CREATE TABLE quotas (
    id                              UUID PRIMARY KEY,
    name                            TEXT NOT NULL,
    max_client_members              INT NOT NULL,
    max_campaigns                   INT NOT NULL,
    max_verifications_per_month     INT NOT NULL,
    max_concurrent_analyses         INT NOT NULL,
    max_file_size_mb                INT NOT NULL,
    report_retention_days           INT NOT NULL,
    allows_video_analysis           BOOLEAN NOT NULL DEFAULT FALSE,
    allows_pdf_export               BOOLEAN NOT NULL DEFAULT FALSE,
    allows_csv_export               BOOLEAN NOT NULL DEFAULT TRUE,
    allows_api_access               BOOLEAN NOT NULL DEFAULT FALSE,
    overage_price_cents             INT NOT NULL DEFAULT 0,
    analysis_priority               INT NOT NULL DEFAULT 0,
    created_at                      TIMESTAMPTZ NOT NULL,
    updated_at                      TIMESTAMPTZ NOT NULL
);

CREATE TABLE plans (
    id                UUID PRIMARY KEY,
    name              TEXT NOT NULL,
    description       TEXT NOT NULL,
    slug              TEXT NOT NULL UNIQUE,
    stripe_price_id   TEXT NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    billing_interval  TEXT NOT NULL,
    price             NUMERIC(10, 2) NOT NULL,
    currency          TEXT NOT NULL,
    quota_id          UUID NOT NULL REFERENCES quotas(id),
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_plans_billing_interval CHECK (billing_interval IN ('month', 'year')),
    CONSTRAINT chk_plans_currency CHECK (currency IN ('EUR', 'USD'))
);

CREATE INDEX IF NOT EXISTS idx_plans_quota_id ON plans (quota_id);
CREATE INDEX IF NOT EXISTS idx_plans_is_active ON plans (is_active);

INSERT INTO quotas (
    id,
    name,
    max_client_members,
    max_campaigns,
    max_verifications_per_month,
    max_concurrent_analyses,
    max_file_size_mb,
    report_retention_days,
    allows_video_analysis,
    allows_pdf_export,
    allows_csv_export,
    allows_api_access,
    overage_price_cents,
    analysis_priority,
    created_at,
    updated_at
) VALUES
(
    '99f2f76e-dc5c-4131-abf6-0e5e307945fb',
    'Free',
    2, 3, 20, 1,
    50, 7,
    FALSE, FALSE, TRUE, FALSE,
    0, 0,
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    '1a4a542b-d9ce-40f3-a534-db8e2d24b039',
    'Starter',
    5, 15, 200, 2,
    200, 30,
    FALSE, TRUE, TRUE, FALSE,
    25, 2,
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    '4362cd30-4f10-4c51-9178-e1abe48fce9c',
    'Pro',
    15, 50, 1000, 5,
    500, 90,
    TRUE, TRUE, TRUE, FALSE,
    15, 5,
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    'cfbafc19-0402-4300-8bff-0b6876b362ee',
    'Business',
    50, 200, 5000, 15,
    2000, 365,
    TRUE, TRUE, TRUE, FALSE,
    10, 10,
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
);

INSERT INTO plans (
    id,
    name,
    description,
    slug,
    stripe_price_id,
    is_active,
    billing_interval,
    price,
    currency,
    quota_id,
    created_at,
    updated_at
) VALUES
(
    '889ea757-54d8-47ee-b8cd-7fece66a0a04',
    'Free',
    'Pour tester le vetting de contenu sur une petite campagne.',
    'free',
    'price_1TwdhH5vcjninxNhcRf8qkUH',
    TRUE,
    'month',
    0.00,
    'EUR',
    '99f2f76e-dc5c-4131-abf6-0e5e307945fb',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    '96d9a6e3-520d-464d-8b87-8dfb11ee082c',
    'Starter',
    'Pour les petites agences qui vérifient leurs campagnes régulièrement.',
    'starter-monthly',
    'price_1Twdhe5vcjninxNhbAVuBcEG',
    TRUE,
    'month',
    12.00,
    'EUR',
    '1a4a542b-d9ce-40f3-a534-db8e2d24b039',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    'e7086b33-92dc-4c68-8f89-156bb9c9316d',
    'Starter',
    'Starter avec 2 mois offerts en facturation annuelle.',
    'starter-yearly',
    'price_1TwdiA5vcjninxNhk4J7n7Tt',
    TRUE,
    'year',
    115.00,
    'EUR',
    '1a4a542b-d9ce-40f3-a534-db8e2d24b039',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    'b24499b5-fb16-4124-9cf8-91d2248689a4',
    'Pro',
    'Pour les agences qui vérifient photo et vidéo en volume, avec export de preuve.',
    'pro-monthly',
    'price_1TwdjN5vcjninxNh3vLqLC6W',
    TRUE,
    'month',
    39.00,
    'EUR',
    '4362cd30-4f10-4c51-9178-e1abe48fce9c',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    'f72156bf-e7a5-47d6-be5c-a2aafe781431',
    'Pro',
    'Pro avec 2 mois offerts en facturation annuelle.',
    'pro-yearly',
    'price_1Twdjd5vcjninxNhS2kRx2WH',
    TRUE,
    'year',
    375.00,
    'EUR',
    '4362cd30-4f10-4c51-9178-e1abe48fce9c',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    '2f8779e7-a08f-4a75-a7b8-0a1f4e9bae8e',
    'Business',
    'Pour les organisations avec des volumes élevés et une rétention étendue des rapports.',
    'business-monthly',
    'price_1Twdjz5vcjninxNhLbKezUvo',
    TRUE,
    'month',
    149.00,
    'EUR',
    'cfbafc19-0402-4300-8bff-0b6876b362ee',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
),
(
    '7db3b043-7298-472e-8b1f-7695010bd733',
    'Business',
    'Business avec 2 mois offerts en facturation annuelle.',
    'business-yearly',
    'price_1TwdkH5vcjninxNhCY6mZyvV',
    TRUE,
    'year',
    1430.00,
    'EUR',
    'cfbafc19-0402-4300-8bff-0b6876b362ee',
    '2026-01-01T00:00:00Z',
    '2026-01-01T00:00:00Z'
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS quotas;

-- +goose StatementEnd
