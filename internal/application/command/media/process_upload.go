package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"

	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type ProcessUploadCommand struct {
	ObjectKey   string
	ContentType string
	Size        int64
}

type ProcessUploadHandler struct {
	mediaRepo   domainmedia.MediaWriteRepository
	contentRepo domaincontent.ContentWriteRepository
	outbox      port.OutboxRepository
	storage     port.Storage
	thumbnailer port.Thumbnailer
}

func NewProcessUploadHandler(
	mediaRepo domainmedia.MediaWriteRepository,
	contentRepo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	thumbnailer port.Thumbnailer,
) *ProcessUploadHandler {
	return &ProcessUploadHandler{
		mediaRepo:   mediaRepo,
		contentRepo: contentRepo,
		outbox:      outbox,
		storage:     storage,
		thumbnailer: thumbnailer,
	}
}

func (h *ProcessUploadHandler) Handle(ctx context.Context, cmd ProcessUploadCommand) error {
	objectKey := strings.TrimSpace(cmd.ObjectKey)
	if objectKey == "" || !domainmedia.IsMediaObjectKey(objectKey) {
		return nil
	}

	media, err := h.mediaRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return errors.New("failed to get media")
	}
	if media == nil {
		return nil
	}
	if media.Status != domainmedia.StatusPendingUpload {
		return nil
	}

	maxBytes := domainmedia.MaxBytesFor(media.MediaType)
	if cmd.Size > maxBytes {
		return h.fail(ctx, media, objectKey, errors.New("file too large"))
	}

	body, err := h.storage.Get(ctx, objectKey)
	if err != nil {
		return h.fail(ctx, media, objectKey, err)
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return h.fail(ctx, media, objectKey, err)
	}
	if int64(len(raw)) > maxBytes {
		return h.fail(ctx, media, objectKey, errors.New("file too large"))
	}

	contentType := strings.TrimSpace(cmd.ContentType)
	if contentType == "" {
		contentType = media.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
		ext := strings.TrimPrefix(filepath.Ext(objectKey), ".")
		if media.MediaType == domainmedia.MediaTypeImage {
			contentType = "image/" + ext
		} else if media.MediaType == domainmedia.MediaTypeVideo {
			contentType = "video/" + ext
		}
	}

	size := cmd.Size
	if size <= 0 {
		size = int64(len(raw))
	}

	switch media.MediaType {
	case domainmedia.MediaTypeImage:
		return h.handleImage(ctx, media, objectKey, contentType, size, raw)
	case domainmedia.MediaTypeVideo:
		return h.handleVideo(ctx, media, objectKey, contentType, size)
	default:
		return h.fail(ctx, media, objectKey, domainmedia.ErrInvalidMediaType)
	}
}

func (h *ProcessUploadHandler) handleImage(
	ctx context.Context,
	media *domainmedia.Media,
	objectKey, contentType string,
	size int64,
	raw []byte,
) error {
	thumbBytes, err := h.thumbnailer.GenerateJPEG(
		ctx,
		bytes.NewReader(raw),
		domainmedia.ThumbnailMaxWidth,
	)
	if err != nil {
		return h.fail(ctx, media, objectKey, err)
	}

	thumbKey := domainmedia.NewThumbnailKey(media.ClientID, media.CampaignID, media.ID)
	if err := h.storage.PutThumbnail(
		ctx,
		thumbKey,
		bytes.NewReader(thumbBytes),
		int64(len(thumbBytes)),
		"image/jpeg",
	); err != nil {
		return h.fail(ctx, media, objectKey, err)
	}

	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return errors.New("failed to get media")
		}
		if fresh == nil {
			return errors.New("media not found")
		}
		if fresh.Status != domainmedia.StatusPendingUpload || fresh.ObjectKey != objectKey {
			return nil
		}

		if err := fresh.MarkUploaded(size, contentType); err != nil {
			return err
		}
		if err := fresh.StartProcessing(); err != nil {
			return err
		}

		content, err := domaincontent.NewFromMedia(
			fresh.ID,
			fresh.CampaignID,
			fresh.ClientID,
			fresh.ObjectKey,
			nil,
			nil,
			size,
			thumbKey,
		)
		if err != nil {
			return err
		}

		if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
			return errors.New("failed to update media")
		}
		if err := h.contentRepo.Save(txCtx, content); err != nil {
			return errors.New("failed to create content")
		}

		events := append(fresh.PullEvents(), content.PullEvents()...)
		return h.outbox.StoreEvents(txCtx, events)
	})
}

func (h *ProcessUploadHandler) handleVideo(
	ctx context.Context,
	media *domainmedia.Media,
	objectKey, contentType string,
	size int64,
) error {
	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return errors.New("failed to get media")
		}
		if fresh == nil {
			return errors.New("media not found")
		}
		if fresh.Status != domainmedia.StatusPendingUpload || fresh.ObjectKey != objectKey {
			return nil
		}

		if err := fresh.MarkUploaded(size, contentType); err != nil {
			return err
		}
		if err := fresh.StartProcessing(); err != nil {
			return err
		}
		if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
			return errors.New("failed to update media")
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
}

func (h *ProcessUploadHandler) fail(
	ctx context.Context,
	media *domainmedia.Media,
	objectKey string,
	cause error,
) error {
	log.Printf(
		"media upload process failed mediaId=%s key=%s: %v",
		media.ID,
		objectKey,
		cause,
	)
	_ = h.storage.Delete(ctx, objectKey)

	err := h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		if fresh.ObjectKey != objectKey || fresh.Status != domainmedia.StatusPendingUpload {
			return nil
		}
		if err := fresh.MarkFailed(cause.Error()); err != nil {
			return err
		}
		if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
	if err != nil {
		return err
	}
	return cause
}
