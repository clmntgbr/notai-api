-- +goose Up
-- +goose StatementBegin
DELETE FROM activity_events
WHERE type = 'campaign.created'
  AND (
    COALESCE(payload->>'name', '') = 'Default'
    OR EXISTS (
        SELECT 1
        FROM campaigns c
        WHERE c.id::text = activity_events.payload->>'campaignId'
          AND c.is_default = TRUE
    )
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
