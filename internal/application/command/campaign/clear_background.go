package campaign

import (
	"context"
	"errors"
	"log"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ClearBackgroundCommand struct {
	CampaignID uuid.UUID
	ClientID   uuid.UUID
}

type ClearBackgroundHandler struct {
	repo    domaincampaign.CampaignWriteRepository
	outbox  port.OutboxRepository
	storage port.Storage
}

func NewClearBackgroundHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
) *ClearBackgroundHandler {
	return &ClearBackgroundHandler{repo: repo, outbox: outbox, storage: storage}
}

func (h *ClearBackgroundHandler) Handle(ctx context.Context, cmd ClearBackgroundCommand) error {
	var pendingKey, thumbnailKey string

	err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.repo.GetByID(txCtx, cmd.CampaignID)
		if err != nil {
			return errors.New("failed to get campaign")
		}
		if campaign == nil || campaign.ClientID != cmd.ClientID {
			return errors.New("campaign not found")
		}

		pendingKey = campaign.BackgroundPendingKey
		thumbnailKey = campaign.BackgroundThumbnailKey

		campaign.ClearBackground()
		if err := h.repo.Update(txCtx, campaign); err != nil {
			return errors.New("failed to update campaign")
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
	if err != nil {
		return err
	}

	if pendingKey != "" {
		if err := h.storage.Delete(ctx, pendingKey); err != nil {
			log.Printf("campaign background: failed to delete pending key=%s: %v", pendingKey, err)
		}
	}
	if thumbnailKey != "" {
		if err := h.storage.DeleteThumbnail(ctx, thumbnailKey); err != nil {
			log.Printf("campaign background: failed to delete thumbnail key=%s: %v", thumbnailKey, err)
		}
	}

	return nil
}
