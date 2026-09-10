package campaign

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const presignExpiry = 15 * time.Minute

type PresignBackgroundCommand struct {
	CampaignID  uuid.UUID
	ClientID    uuid.UUID
	Filename    string
	ContentType string
}

type PresignBackgroundResult struct {
	URL string
}

type PresignBackgroundHandler struct {
	repo    domaincampaign.CampaignWriteRepository
	outbox  port.OutboxRepository
	storage port.Storage
}

func NewPresignBackgroundHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
) *PresignBackgroundHandler {
	return &PresignBackgroundHandler{repo: repo, outbox: outbox, storage: storage}
}

func (h *PresignBackgroundHandler) Handle(
	ctx context.Context,
	cmd PresignBackgroundCommand,
) (*PresignBackgroundResult, error) {
	filename := filepath.Base(strings.TrimSpace(cmd.Filename))
	if err := domaincampaign.ValidateBackgroundFilename(filename); err != nil {
		return nil, err
	}

	pendingKey := domaincampaign.NewBackgroundObjectKey(cmd.ClientID, cmd.CampaignID, filename)
	contentType := strings.TrimSpace(cmd.ContentType)

	var previousPendingKey string

	err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.repo.GetByID(txCtx, cmd.CampaignID)
		if err != nil {
			return errors.New("failed to get campaign")
		}
		if campaign == nil {
			return errors.New("campaign not found")
		}
		if campaign.ClientID != cmd.ClientID {
			return errors.New("campaign not found")
		}

		previousPendingKey = campaign.BackgroundPendingKey
		campaign.StartBackgroundUpload(pendingKey, filename, contentType)
		if err := h.repo.Update(txCtx, campaign); err != nil {
			return errors.New("failed to update campaign")
		}
		return h.outbox.StoreEvents(txCtx, campaign.PullEvents())
	})
	if err != nil {
		return nil, err
	}

	if previousPendingKey != "" && previousPendingKey != pendingKey {
		_ = h.storage.Delete(ctx, previousPendingKey)
	}

	url, err := h.storage.PresignedPutURL(ctx, pendingKey, presignExpiry)
	if err != nil {
		return nil, errors.New("failed to generate upload url")
	}

	return &PresignBackgroundResult{URL: url}, nil
}
