package campaign

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"
)

type ProcessBackgroundUploadCommand struct {
	ObjectKey   string
	ContentType string
	Size        int64
}

type ProcessBackgroundUploadHandler struct {
	repo        domaincampaign.CampaignWriteRepository
	outbox      port.OutboxRepository
	storage     port.Storage
	thumbnailer port.Thumbnailer
}

func NewProcessBackgroundUploadHandler(
	repo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	thumbnailer port.Thumbnailer,
) *ProcessBackgroundUploadHandler {
	return &ProcessBackgroundUploadHandler{
		repo:        repo,
		outbox:      outbox,
		storage:     storage,
		thumbnailer: thumbnailer,
	}
}

func (h *ProcessBackgroundUploadHandler) Handle(
	ctx context.Context,
	cmd ProcessBackgroundUploadCommand,
) error {
	objectKey := strings.TrimSpace(cmd.ObjectKey)
	if objectKey == "" || domaincampaign.IsThumbnailObjectKey(objectKey) {
		return nil
	}

	campaign, err := h.repo.GetByBackgroundPendingKey(ctx, objectKey)
	if err != nil {
		return errors.New("failed to get campaign")
	}
	if campaign == nil {
		return nil
	}

	if cmd.Size > domaincampaign.MaxBackgroundBytes {
		return h.fail(ctx, campaign, objectKey, errors.New("file too large"))
	}

	body, err := h.storage.Get(ctx, objectKey)
	if err != nil {
		return h.fail(ctx, campaign, objectKey, err)
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, domaincampaign.MaxBackgroundBytes+1))
	if err != nil {
		return h.fail(ctx, campaign, objectKey, err)
	}
	if int64(len(raw)) > domaincampaign.MaxBackgroundBytes {
		return h.fail(ctx, campaign, objectKey, errors.New("file too large"))
	}

	thumbBytes, err := h.thumbnailer.GenerateJPEG(
		ctx,
		bytes.NewReader(raw),
		domaincampaign.ThumbnailMaxWidth,
	)
	if err != nil {
		return h.fail(ctx, campaign, objectKey, err)
	}

	thumbKey := domaincampaign.NewBackgroundThumbnailKey(campaign.ClientID, campaign.ID)
	if err := h.storage.PutThumbnail(
		ctx,
		thumbKey,
		bytes.NewReader(thumbBytes),
		int64(len(thumbBytes)),
		"image/jpeg",
	); err != nil {
		return h.fail(ctx, campaign, objectKey, err)
	}

	if err := h.storage.Delete(ctx, objectKey); err != nil {
		log.Printf("campaign background: failed to delete original key=%s: %v", objectKey, err)
	}

	contentType := strings.TrimSpace(cmd.ContentType)
	if contentType == "" {
		contentType = campaign.BackgroundContentType
	}
	if contentType == "" {
		contentType = "image/" + strings.TrimPrefix(filepath.Ext(objectKey), ".")
	}

	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.repo.GetByID(txCtx, campaign.ID)
		if err != nil {
			return errors.New("failed to get campaign")
		}
		if fresh == nil {
			return errors.New("campaign not found")
		}
		if fresh.BackgroundPendingKey != objectKey {
			return nil
		}

		fresh.ApplyBackgroundReady(thumbKey, contentType)
		if err := h.repo.Update(txCtx, fresh); err != nil {
			return errors.New("failed to update campaign")
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
}

func (h *ProcessBackgroundUploadHandler) fail(
	ctx context.Context,
	campaign *domaincampaign.Campaign,
	objectKey string,
	cause error,
) error {
	log.Printf(
		"campaign background process failed campaignId=%s key=%s: %v",
		campaign.ID,
		objectKey,
		cause,
	)
	_ = h.storage.Delete(ctx, objectKey)

	err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.repo.GetByID(txCtx, campaign.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		if fresh.BackgroundPendingKey != "" && fresh.BackgroundPendingKey != objectKey {
			return nil
		}
		fresh.MarkBackgroundFailed()
		if err := h.repo.Update(txCtx, fresh); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
	if err != nil {
		return err
	}
	return cause
}
