package campaign

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type DeleteCampaignCommand struct {
	ID uuid.UUID
}

type DeleteCampaignHandler struct {
	repo   domaincampaign.CampaignWriteRepository
	outbox port.OutboxRepository
}

func NewDeleteCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
) *DeleteCampaignHandler {
	return &DeleteCampaignHandler{repo: repo, outbox: outbox}
}

func (h *DeleteCampaignHandler) Handle(ctx context.Context, cmd DeleteCampaignCommand) error {
	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.repo.GetByID(txCtx, cmd.ID)
		if err != nil {
			return errors.New("failed to get campaign")
		}
		if campaign == nil {
			return errors.New("campaign not found")
		}

		if err := campaign.SoftDelete(); err != nil {
			return err
		}

		if err := h.repo.Update(txCtx, campaign); err != nil {
			return errors.New("failed to delete campaign")
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
}
