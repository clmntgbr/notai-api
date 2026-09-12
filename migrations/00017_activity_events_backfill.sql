-- +goose Up
-- +goose StatementBegin

-- Backfill activity feed from existing outbox events (idempotent on activity id = event id).

INSERT INTO activity_events (
    id, client_id, type, actor_type, actor_user_id, actor_name, message, payload, occurred_at, created_at
)
SELECT
    o.id,
    (o.payload->>'clientId')::uuid,
    CASE o.payload->>'label'
        WHEN 'ai_generated' THEN 'content.ai_flagged'
        WHEN 'uncertain' THEN 'content.manual_review'
    END,
    'system',
    NULL,
    'Système',
    CASE o.payload->>'label'
        WHEN 'ai_generated' THEN format(
            '« %s » signalé comme généré par IA (score %s %%)',
            COALESCE(NULLIF(c.filename, ''), o.payload->>'contentId'),
            ROUND(((o.payload->>'confidence')::double precision) * 100)::int
        )
        WHEN 'uncertain' THEN format(
            '« %s » placé en revue manuelle (score %s %%)',
            COALESCE(NULLIF(c.filename, ''), o.payload->>'contentId'),
            ROUND(((o.payload->>'confidence')::double precision) * 100)::int
        )
    END,
    jsonb_build_object(
        'contentId', o.payload->>'contentId',
        'campaignId', o.payload->>'campaignId',
        'filename', COALESCE(c.filename, ''),
        'label', o.payload->>'label',
        'confidence', (o.payload->>'confidence')::double precision
    ),
    COALESCE(o.created_at, NOW()),
    NOW()
FROM outbox_events o
LEFT JOIN contents c ON c.id::text = o.payload->>'contentId'
WHERE o.event_type = 'content.verdict_rendered.v1'
  AND o.payload->>'label' IN ('ai_generated', 'uncertain')
  AND o.payload->>'clientId' IS NOT NULL
ON CONFLICT (id) DO NOTHING;

INSERT INTO activity_events (
    id, client_id, type, actor_type, actor_user_id, actor_name, message, payload, occurred_at, created_at
)
SELECT
    o.id,
    (o.payload->>'clientId')::uuid,
    'campaign.created',
    'system',
    NULL,
    'Système',
    format(
        'Nouvelle campagne « %s » créée pour %s',
        COALESCE(o.payload->>'name', 'Sans nom'),
        COALESCE(cl.name, 'le client')
    ),
    jsonb_build_object(
        'campaignId', o.payload->>'campaignId',
        'name', o.payload->>'name',
        'clientName', COALESCE(cl.name, '')
    ),
    COALESCE(o.created_at, NOW()),
    NOW()
FROM outbox_events o
LEFT JOIN clients cl ON cl.id::text = o.payload->>'clientId'
LEFT JOIN campaigns ca ON ca.id::text = o.payload->>'campaignId'
WHERE o.event_type = 'campaign.created.v1'
  AND COALESCE((o.payload->>'isDefault')::boolean, FALSE) = FALSE
  AND COALESCE(ca.is_default, FALSE) = FALSE
  AND o.payload->>'clientId' IS NOT NULL
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Keep projected rows; down of 00016 drops the table.
SELECT 1;
-- +goose StatementEnd
