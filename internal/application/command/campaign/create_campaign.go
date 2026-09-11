package campaign

import (
	"context"
	"errors"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type CreateCampaignCommand struct {
	Name      string
	ClientID  uuid.UUID
	StartAt *time.Time
	EndAt   *time.Time
}

type CreateCampaignHandler struct {
	repo   domaincampaign.CampaignWriteRepository
	outbox port.OutboxRepository
}

func NewCreateCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
) *CreateCampaignHandler {
	return &CreateCampaignHandler{repo: repo, outbox: outbox}
}

func (h *CreateCampaignHandler) Handle(
	ctx context.Context,
	cmd CreateCampaignCommand,
) (*domaincampaign.Campaign, error) {
	if cmd.Name == "" {
		return nil, errors.New("name is required")
	}
	if cmd.ClientID == uuid.Nil {
		return nil, errors.New("clientId is required")
	}

	campaign, err := domaincampaign.NewCampaign(cmd.Name, cmd.ClientID, cmd.StartAt, cmd.EndAt)
	if err != nil {
		return nil, err
	}

	err = h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.repo.Save(txCtx, campaign); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
	if err != nil {
		if errors.Is(err, domaincampaign.ErrInvalidSchedule) {
			return nil, err
		}
		return nil, errors.New("failed to create campaign")
	}

	return campaign, nil
}
