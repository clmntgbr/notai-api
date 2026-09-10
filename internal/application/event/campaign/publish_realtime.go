package campaign

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type PublishRealtimeHandler struct {
	publisher  *realtime.Publisher
	clientRepo domainclient.ClientReadRepository
}

func NewPublishRealtimeHandler(
	realtimePublisher port.RealtimePublisher,
	clientRepo domainclient.ClientReadRepository,
) *PublishRealtimeHandler {
	return &PublishRealtimeHandler{
		publisher:  realtime.NewPublisher(realtimePublisher),
		clientRepo: clientRepo,
	}
}

func (h *PublishRealtimeHandler) OnCreated(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionCreated, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnUpdated(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionUpdated, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnDeleted(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionDeleted, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnBackgroundUpdated(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignBackgroundUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionBackgroundUpdated, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) publishToClientMembers(
	ctx context.Context,
	action string,
	clientIDRaw string,
	payload any,
) error {
	clientID, err := uuid.Parse(clientIDRaw)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	view, err := h.clientRepo.FindByID(ctx, clientID)
	if err != nil {
		return messaging.Retryable(err)
	}
	if view == nil {
		return nil
	}
	return h.publisher.ToMembers(ctx, realtime.EntityCampaign, action, view.MemberIDs, payload)
}
