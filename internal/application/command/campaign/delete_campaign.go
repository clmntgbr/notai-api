package campaign

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type DeleteCampaignCommand struct {
	ID uuid.UUID
}

type DeleteCampaignHandler struct {
	repo        domaincampaign.CampaignWriteRepository
	mediaRepo   domainmedia.MediaWriteRepository
	contentRepo domaincontent.ContentWriteRepository
	outbox      port.OutboxRepository
}

func NewDeleteCampaignHandler(
	repo domaincampaign.CampaignWriteRepository,
	mediaRepo domainmedia.MediaWriteRepository,
	contentRepo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
) *DeleteCampaignHandler {
	return &DeleteCampaignHandler{
		repo:        repo,
		mediaRepo:   mediaRepo,
		contentRepo: contentRepo,
		outbox:      outbox,
	}
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

		medias, err := h.mediaRepo.ListActiveByCampaignID(txCtx, campaign.ID)
		if err != nil {
			return errors.New("failed to list campaign medias")
		}

		events := campaign.PullEvents()
		for _, media := range medias {
			refs, err := h.contentObjectRefs(txCtx, media.ID)
			if err != nil {
				return err
			}
			media.SoftDelete(refs)
			if err := h.mediaRepo.Update(txCtx, media); err != nil {
				return errors.New("failed to delete campaign medias")
			}
			events = append(events, media.PullEvents()...)
		}

		return h.outbox.StoreEvents(txCtx, events)
	})
}

func (h *DeleteCampaignHandler) contentObjectRefs(
	ctx context.Context,
	mediaID uuid.UUID,
) ([]domainmedia.SoftDeleteObjectRef, error) {
	contents, err := h.contentRepo.ListByMediaID(ctx, mediaID)
	if err != nil {
		return nil, errors.New("failed to list media contents")
	}
	refs := make([]domainmedia.SoftDeleteObjectRef, 0, len(contents))
	for _, content := range contents {
		thumb := ""
		if content.ThumbnailKey != nil {
			thumb = *content.ThumbnailKey
		}
		refs = append(refs, domainmedia.SoftDeleteObjectRef{
			ObjectKey:    content.ObjectKey,
			ThumbnailKey: thumb,
		})
	}
	return refs, nil
}
