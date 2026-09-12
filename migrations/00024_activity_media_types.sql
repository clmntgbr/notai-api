-- +goose Up
-- +goose StatementBegin
ALTER TABLE activity_events DROP CONSTRAINT IF EXISTS activity_events_type_check;
ALTER TABLE activity_events
    ADD CONSTRAINT activity_events_type_check CHECK (
        type IN (
            'content.ai_flagged',
            'content.manual_review',
            'content.human_verified',
            'content.failed',
            'media.ai_flagged',
            'media.manual_review',
            'media.human_verified',
            'media.failed',
            'campaign.created',
            'client.member_added'
        )
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE activity_events DROP CONSTRAINT IF EXISTS activity_events_type_check;
ALTER TABLE activity_events
    ADD CONSTRAINT activity_events_type_check CHECK (
        type IN (
            'content.ai_flagged',
            'content.manual_review',
            'content.human_verified',
            'content.failed',
            'campaign.created',
            'client.member_added'
        )
    );
-- +goose StatementEnd
