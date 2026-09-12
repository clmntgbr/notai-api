-- +goose Up
-- +goose StatementBegin
ALTER TABLE activity_events DROP CONSTRAINT IF EXISTS activity_events_type_check;
ALTER TABLE activity_events
    ADD CONSTRAINT activity_events_type_check CHECK (
        type IN (
            'content.ai_flagged',
            'content.manual_review',
            'content.human_verified',
            'campaign.created',
            'client.member_added'
        )
    );

-- Backfill human verdicts previously skipped by the projector.
INSERT INTO activity_events (
    id, client_id, type, actor_type, actor_user_id, actor_name, message, payload, occurred_at, created_at
)
SELECT
    o.id,
    (o.payload->>'clientId')::uuid,
    'content.human_verified',
    'system',
    NULL,
    'Système',
    format(
        '« %s » validé comme contenu humain (score %s %%)',
        COALESCE(NULLIF(c.filename, ''), o.payload->>'contentId'),
        ROUND(((o.payload->>'confidence')::double precision) * 100)::int
    ),
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
  AND o.payload->>'label' = 'human'
  AND o.payload->>'clientId' IS NOT NULL
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM activity_events WHERE type = 'content.human_verified';

ALTER TABLE activity_events DROP CONSTRAINT IF EXISTS activity_events_type_check;
ALTER TABLE activity_events
    ADD CONSTRAINT activity_events_type_check CHECK (
        type IN (
            'content.ai_flagged',
            'content.manual_review',
            'campaign.created',
            'client.member_added'
        )
    );
-- +goose StatementEnd
