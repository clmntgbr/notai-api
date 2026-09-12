-- +goose Up
-- +goose StatementBegin

-- Rewrite existing French activity copy to English.
UPDATE activity_events SET actor_name = 'System' WHERE actor_name = 'Système';

UPDATE activity_events a
SET message = format(
    '“%s” flagged as AI-generated (score %s%%)',
    COALESCE(NULLIF(a.payload->>'filename', ''), a.payload->>'contentId', 'file'),
    ROUND(COALESCE((a.payload->>'confidence')::double precision, 0) * 100)::int
)
WHERE a.type = 'content.ai_flagged';

UPDATE activity_events a
SET message = format(
    '“%s” placed under manual review (score %s%%)',
    COALESCE(NULLIF(a.payload->>'filename', ''), a.payload->>'contentId', 'file'),
    ROUND(COALESCE((a.payload->>'confidence')::double precision, 0) * 100)::int
)
WHERE a.type = 'content.manual_review';

UPDATE activity_events a
SET message = format(
    '“%s” verified as human content (score %s%%)',
    COALESCE(NULLIF(a.payload->>'filename', ''), a.payload->>'contentId', 'file'),
    ROUND(COALESCE((a.payload->>'confidence')::double precision, 0) * 100)::int
)
WHERE a.type = 'content.human_verified';

UPDATE activity_events a
SET message = format(
    '“%s” failed during processing',
    COALESCE(NULLIF(a.payload->>'filename', ''), a.payload->>'contentId', 'file')
)
WHERE a.type = 'content.failed';

UPDATE activity_events a
SET message = format(
    'New campaign “%s” created for %s',
    COALESCE(NULLIF(a.payload->>'name', ''), 'Untitled'),
    COALESCE(NULLIF(a.payload->>'clientName', ''), 'the client')
)
WHERE a.type = 'campaign.created';

UPDATE activity_events a
SET message = format(
    '%s joined the team',
    COALESCE(NULLIF(a.payload->>'memberName', ''), 'A member')
)
WHERE a.type = 'client.member_added';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
