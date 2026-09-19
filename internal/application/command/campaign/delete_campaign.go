package campaign

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type DeleteCampaignCommand struct {
	ID uuid.UUID
}

type DeleteCampaignHandler struct {
	repo      domaincampaign.CampaignWriteRepository
	mediaRepo domainmedia.MediaWriteRepository
	outbox    port.OutboxRepository
}

func NewDeleteCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	mediaRepo domainmedia.MediaWriteRepository,
	outbox port.OutboxRepository,
) *DeleteCampaignHandler {
	return &DeleteCampaignHandler{repo: repo, mediaRepo: mediaRepo, outbox: outbox}
}

func (h *DeleteCampaignHandler) Handle(ctx context.Context, cmd DeleteCampaignCommand) error {
	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.repo.GetByIDForUpdate(txCtx, cmd.ID)
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
		if err := h.mediaRepo.SoftDeleteByCampaignID(txCtx, campaign.ID); err != nil {
			return errors.New("failed to delete campaign medias")
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
}
