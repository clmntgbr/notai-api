package content

import (
	"context"
	"encoding/json"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domainclient "go-api/internal/domain/client"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// PublishRealtimeHandler publishes only terminal content verdicts.
// Intermediate analysis steps (created / uploaded / status_changed / detector
// result rows) stay in the outbox/DB for the pipeline and are not mirrored to
// Centrifugo — the agency UI reacts to media-level progress plus this final
// per-content verdict when drilling into a media.
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

func (h *PublishRealtimeHandler) OnVerdictRendered(ctx context.Context, payload []byte) error {
	var evt domaincontent.ContentVerdictRendered
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	return h.publishToClientMembers(ctx, realtime.ActionVerdictRendered, evt.ClientID, evt)
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
	return h.publisher.ToMembers(ctx, realtime.EntityContent, action, view.MemberIDs, payload)
}
