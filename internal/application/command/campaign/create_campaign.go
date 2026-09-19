package campaign

import (
	"context"
	"errors"
	"time"

	cmdquota "go-api/internal/application/command/quota"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type CreateCampaignCommand struct {
	Name     string
	ClientID uuid.UUID
	StartAt  *time.Time
	EndAt    *time.Time
}

type CreateCampaignHandler struct {
	repo   domaincampaign.CampaignWriteRepository
	outbox port.OutboxRepository
	quota  *cmdquota.AssertCreateAllowedHandler
}

func NewCreateCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
	quota *cmdquota.AssertCreateAllowedHandler,
) *CreateCampaignHandler {
	return &CreateCampaignHandler{repo: repo, outbox: outbox, quota: quota}
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

	if h.quota != nil {
		if err := h.quota.AssertCampaignCreate(ctx, cmd.ClientID); err != nil {
			return nil, err
		}
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
		return nil, errors.New("failed to create campaign")
	}

	return campaign, nil
}
