-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS activity_events (
    id UUID PRIMARY KEY,
    client_id UUID NOT NULL REFERENCES clients (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    actor_name TEXT NOT NULL,
    message TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT activity_events_actor_type_check CHECK (
        actor_type IN ('system', 'user')
    ),
    CONSTRAINT activity_events_type_check CHECK (
        type IN (
            'content.ai_flagged',
            'content.manual_review',
            'campaign.created',
            'client.member_added'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_activity_events_client_occurred
    ON activity_events (client_id, occurred_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS activity_events;
-- +goose StatementEnd
