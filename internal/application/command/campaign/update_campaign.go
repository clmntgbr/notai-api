package campaign

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type UpdateCampaignCommand struct {
	ID   uuid.UUID
	Name string
}

type UpdateCampaignHandler struct {
	repo   domaincampaign.CampaignWriteRepository
	outbox port.OutboxRepository
}

func NewUpdateCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
) *UpdateCampaignHandler {
	return &UpdateCampaignHandler{repo: repo, outbox: outbox}
}

func (h *UpdateCampaignHandler) Handle(ctx context.Context, cmd UpdateCampaignCommand) error {
	if cmd.Name == "" {
		return errors.New("name is required")
	}

	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.repo.GetByID(txCtx, cmd.ID)
		if err != nil {
			return errors.New("failed to get campaign")
		}
		if campaign == nil {
			return errors.New("campaign not found")
		}

		if err := campaign.ApplyUpdate(cmd.Name); err != nil {
			return err
		}

		if err := h.repo.Update(txCtx, campaign); err != nil {
			return errors.New("failed to update campaign")
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
}
