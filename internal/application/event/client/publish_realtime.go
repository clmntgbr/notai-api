package client

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	"go-api/internal/domain/port"
	domainclient "go-api/internal/domain/client"
)

type PublishRealtimeHandler struct {
	publisher *realtime.Publisher
}

func NewPublishRealtimeHandler(realtimePublisher port.RealtimePublisher) *PublishRealtimeHandler {
	return &PublishRealtimeHandler{
		publisher: realtime.NewPublisher(realtimePublisher),
	}
}

func (h *PublishRealtimeHandler) OnCreated(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToUser(ctx, realtime.EntityClient, realtime.ActionCreated, evt.CreatedByUserID, evt)
}

func (h *PublishRealtimeHandler) OnUpdated(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToUsers(ctx, realtime.EntityClient, realtime.ActionUpdated, evt.MemberIDs, evt)
}

func (h *PublishRealtimeHandler) OnDeleted(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToUsers(ctx, realtime.EntityClient, realtime.ActionDeleted, evt.MemberIDs, evt)
}

func (h *PublishRealtimeHandler) OnMemberAdded(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientMemberAdded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToUser(ctx, realtime.EntityClient, realtime.ActionMemberAdded, evt.UserID, evt)
}

func (h *PublishRealtimeHandler) OnMemberRemoved(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientMemberRemoved
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publisher.ToUser(ctx, realtime.EntityClient, realtime.ActionMemberRemoved, evt.UserID, evt)
}
