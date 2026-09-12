package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/event"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ExtractFramesCommand struct {
	MediaID uuid.UUID
}

type ExtractFramesHandler struct {
	mediaRepo    domainmedia.MediaWriteRepository
	contentRepo  domaincontent.ContentWriteRepository
	outbox       port.OutboxRepository
	storage      port.Storage
	extractor    port.FrameExtractor
	intervalMs   int
	maxFrames    int
}

func NewExtractFramesHandler(
	mediaRepo domainmedia.MediaWriteRepository,
	contentRepo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
	extractor port.FrameExtractor,
	intervalMs, maxFrames int,
) *ExtractFramesHandler {
	if intervalMs <= 0 {
		intervalMs = 2000
	}
	if maxFrames <= 0 {
		maxFrames = 12
	}
	return &ExtractFramesHandler{
		mediaRepo:   mediaRepo,
		contentRepo: contentRepo,
		outbox:      outbox,
		storage:     storage,
		extractor:   extractor,
		intervalMs:  intervalMs,
		maxFrames:   maxFrames,
	}
}

func (h *ExtractFramesHandler) Handle(ctx context.Context, cmd ExtractFramesCommand) error {
	if cmd.MediaID == uuid.Nil {
		return errors.New("media id is required")
	}

	media, err := h.mediaRepo.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if media == nil {
		return nil
	}
	if media.MediaType != domainmedia.MediaTypeVideo {
		return nil
	}
	if media.Status != domainmedia.StatusProcessing && media.Status != domainmedia.StatusUploaded {
		return nil
	}

	// Ensure processing status (idempotent).
	if media.Status == domainmedia.StatusUploaded {
		if err := media.StartProcessing(); err != nil {
			return err
		}
		if err := h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := h.mediaRepo.Update(txCtx, media); err != nil {
				return err
			}
			return h.outbox.StoreEvents(txCtx, media.PullEvents())
		}); err != nil {
			return err
		}
	}

	existing, err := h.contentRepo.ListByMediaID(ctx, media.ID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		// Already extracted (retry after partial success publishing).
		return nil
	}

	tmpFile, cleanup, err := h.downloadToTemp(ctx, media.ObjectKey)
	if err != nil {
		return h.failMedia(ctx, media, err)
	}
	defer cleanup()

	frames, err := h.extractor.Extract(ctx, tmpFile, h.intervalMs, h.maxFrames)
	if err != nil {
		return h.failMedia(ctx, media, err)
	}

	var created []*domaincontent.Content
	for _, frame := range frames {
		objectKey := domainmedia.NewFrameObjectKey(
			media.ClientID,
			media.CampaignID,
			media.ID,
			frame.Index,
		)
		if err := h.storage.Put(
			ctx,
			objectKey,
			bytes.NewReader(frame.JPEGBytes),
			int64(len(frame.JPEGBytes)),
			"image/jpeg",
		); err != nil {
			return err // infra retry — do not MarkFailed
		}

		idx := frame.Index
		ts := frame.TimestampMs
		content, err := domaincontent.NewFromMedia(
			media.ID,
			media.CampaignID,
			media.ClientID,
			objectKey,
			&idx,
			&ts,
			int64(len(frame.JPEGBytes)),
			"",
		)
		if err != nil {
			return err
		}
		thumbKey := domaincontent.NewThumbnailKey(media.ClientID, media.CampaignID, content.ID)
		if err := h.storage.PutThumbnail(
			ctx,
			thumbKey,
			bytes.NewReader(frame.JPEGBytes),
			int64(len(frame.JPEGBytes)),
			"image/jpeg",
		); err != nil {
			return err
		}
		content.SetThumbnailKey(thumbKey)
		created = append(created, content)
	}

	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		// Re-check no contents appeared concurrently.
		siblings, err := h.contentRepo.ListByMediaID(txCtx, fresh.ID)
		if err != nil {
			return err
		}
		if len(siblings) > 0 {
			return nil
		}

		var events []event.DomainEvent
		for _, c := range created {
			if err := h.contentRepo.Save(txCtx, c); err != nil {
				return fmt.Errorf("save content frame: %w", err)
			}
			events = append(events, c.PullEvents()...)
		}
		return h.outbox.StoreEvents(txCtx, events)
	})
}

func (h *ExtractFramesHandler) downloadToTemp(ctx context.Context, objectKey string) (string, func(), error) {
	body, err := h.storage.Get(ctx, objectKey)
	if err != nil {
		return "", nil, err
	}
	defer body.Close()

	tmp, err := os.CreateTemp("", "media-video-*"+filepath.Ext(objectKey))
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}

	if _, err := io.Copy(tmp, body); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", nil, err
	}
	return tmp.Name(), func() { _ = os.Remove(tmp.Name()) }, nil
}

func (h *ExtractFramesHandler) failMedia(ctx context.Context, media *domainmedia.Media, cause error) error {
	err := h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, media.ID)
		if err != nil {
			return err
		}
		if fresh == nil {
			return nil
		}
		if fresh.Status == domainmedia.StatusAnalyzed || fresh.Status == domainmedia.StatusFailed {
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
