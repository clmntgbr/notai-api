package media

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domainclient "go-api/internal/domain/client"
	domainmedia "go-api/internal/domain/media"
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

func (h *PublishRealtimeHandler) OnUploadRequested(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaUploadRequested
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionCreated, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnUploaded(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionUpdated, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnStatusChanged(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaStatusChanged
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionStatusChanged, evt.ClientID, evt)
}

func (h *PublishRealtimeHandler) OnVerdictRendered(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaVerdictRendered
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionUpdated, evt.ClientID, evt)
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
	return h.publisher.ToMembers(ctx, realtime.EntityMedia, action, view.MemberIDs, payload)
}
