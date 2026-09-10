package content

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"
)

type ProcessUploadCommand struct {
	ObjectKey   string
	ContentType string
	Size        int64
}

type ProcessUploadHandler struct {
	repo        domaincontent.ContentWriteRepository
	outbox      port.OutboxRepository
	storage     port.Storage
	thumbnailer port.Thumbnailer
}

func NewProcessUploadHandler(
	repo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	thumbnailer port.Thumbnailer,
) *ProcessUploadHandler {
	return &ProcessUploadHandler{
		repo:        repo,
		outbox:      outbox,
		storage:     storage,
		thumbnailer: thumbnailer,
	}
}

func (h *ProcessUploadHandler) Handle(ctx context.Context, cmd ProcessUploadCommand) error {
	objectKey := strings.TrimSpace(cmd.ObjectKey)
	if objectKey == "" || !domaincontent.IsContentObjectKey(objectKey) {
		return nil
	}

	content, err := h.repo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return errors.New("failed to get content")
	}
	if content == nil {
		return nil
	}
	if content.Status != domaincontent.StatusPendingUpload {
		return nil
	}

	if cmd.Size > domaincontent.MaxContentBytes {
		return h.fail(ctx, content, objectKey, errors.New("file too large"))
	}

	body, err := h.storage.Get(ctx, objectKey)
	if err != nil {
		return h.fail(ctx, content, objectKey, err)
	}
	defer body.Close()

	raw, err := io.ReadAll(io.LimitReader(body, domaincontent.MaxContentBytes+1))
	if err != nil {
		return h.fail(ctx, content, objectKey, err)
	}
	if int64(len(raw)) > domaincontent.MaxContentBytes {
		return h.fail(ctx, content, objectKey, errors.New("file too large"))
	}

	thumbBytes, err := h.thumbnailer.GenerateJPEG(
		ctx,
		bytes.NewReader(raw),
		domaincontent.ThumbnailMaxWidth,
	)
	if err != nil {
		return h.fail(ctx, content, objectKey, err)
	}

	thumbKey := domaincontent.NewThumbnailKey(content.ClientID, content.CampaignID, content.ID)
	if err := h.storage.PutThumbnail(
		ctx,
		thumbKey,
		bytes.NewReader(thumbBytes),
		int64(len(thumbBytes)),
		"image/jpeg",
	); err != nil {
		return h.fail(ctx, content, objectKey, err)
	}

	contentType := strings.TrimSpace(cmd.ContentType)
	if contentType == "" {
		contentType = content.ContentType
	}
	if contentType == "" {
		contentType = "image/" + strings.TrimPrefix(filepath.Ext(objectKey), ".")
	}

	size := cmd.Size
	if size <= 0 {
		size = int64(len(raw))
	}

	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.repo.GetByID(txCtx, content.ID)
		if err != nil {
			return errors.New("failed to get content")
		}
		if fresh == nil {
			return errors.New("content not found")
		}
		if fresh.Status != domaincontent.StatusPendingUpload || fresh.ObjectKey != objectKey {
			return nil
		}

		if err := fresh.MarkUploaded(size, contentType, thumbKey); err != nil {
			return err
		}
		if err := h.repo.Update(txCtx, fresh); err != nil {
			return errors.New("failed to update content")
		}
		return h.outbox.StoreEvents(txCtx, fresh.PullEvents())
	})
}

func (h *ProcessUploadHandler) fail(
	ctx context.Context,
	content *domaincontent.Content,
	objectKey string,
	cause error,
) error {
	log.Printf(
		"content upload process failed contentId=%s key=%s: %v",
		content.ID,
		objectKey,
		cause,
	)
	_ = h.storage.Delete(ctx, objectKey)

	err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.repo.GetByID(txCtx, content.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		if fresh.ObjectKey != objectKey || fresh.Status != domaincontent.StatusPendingUpload {
			return nil
		}
		if err := fresh.MarkFailed(); err != nil {
			return err
		}
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
