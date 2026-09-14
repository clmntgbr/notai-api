package media

import (
	"context"
	"fmt"
	"log"
	"time"

	contentcmd "go-api/internal/application/command/content"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/event"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const (
	DefaultStaleAfter = 30 * time.Minute
	DefaultMaxAge     = 2 * time.Hour
	DefaultBatchSize  = 100
	StaleFailReason   = "Media processing timed out"
)

type RecoverStaleProcessingMediasResult struct {
	Scanned  int
	Requeued int
	Failed   int
	Errors   int
}

type RecoverStaleProcessingMediasHandler struct {
	mediaRepo   domainmedia.MediaWriteRepository
	contentRepo domaincontent.ContentWriteRepository
	outbox      port.OutboxRepository
	staleAfter  time.Duration
	maxAge      time.Duration
	batchSize   int
}

func NewRecoverStaleProcessingMediasHandler(
	mediaRepo domainmedia.MediaWriteRepository,
	contentRepo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
	staleAfter, maxAge time.Duration,
) *RecoverStaleProcessingMediasHandler {
	if staleAfter <= 0 {
		staleAfter = DefaultStaleAfter
	}
	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	return &RecoverStaleProcessingMediasHandler{
		mediaRepo:   mediaRepo,
		contentRepo: contentRepo,
		outbox:      outbox,
		staleAfter:  staleAfter,
		maxAge:      maxAge,
		batchSize:   DefaultBatchSize,
	}
}

func (h *RecoverStaleProcessingMediasHandler) Handle(
	ctx context.Context,
	now time.Time,
) (*RecoverStaleProcessingMediasResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	staleBefore := now.Add(-h.staleAfter)
	medias, err := h.mediaRepo.ListProcessingUpdatedBefore(ctx, staleBefore, h.batchSize)
	if err != nil {
		return nil, fmt.Errorf("list stale processing medias: %w", err)
	}

	result := &RecoverStaleProcessingMediasResult{Scanned: len(medias)}
	maxAgeCutoff := now.Add(-h.maxAge)

	for _, mediaEntity := range medias {
		if mediaEntity.CreatedAt.Before(maxAgeCutoff) {
			if err := h.failMedia(ctx, mediaEntity.ID); err != nil {
				log.Printf("scheduler: fail media %s: %v", mediaEntity.ID, err)
				result.Errors++
				continue
			}
			result.Failed++
			continue
		}
		requeued, err := h.requeueMedia(ctx, mediaEntity.ID, staleBefore, now)
		if err != nil {
			log.Printf("scheduler: requeue media %s: %v", mediaEntity.ID, err)
			result.Errors++
			continue
		}
		if requeued {
			result.Requeued++
		}
	}

	return result, nil
}

func (h *RecoverStaleProcessingMediasHandler) failMedia(ctx context.Context, mediaID uuid.UUID) error {
	return h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, mediaID)
		if err != nil {
			return err
		}
		if fresh == nil || fresh.Status != domainmedia.StatusProcessing {
			return nil
		}

		events := make([]event.DomainEvent, 0)
		contents, err := h.contentRepo.ListByMediaID(txCtx, fresh.ID)
		if err != nil {
			return err
		}
		for i := range contents {
			c := &contents[i]
			if c.IsTerminal() {
				continue
			}
			if err := c.MarkFailed(); err != nil {
				return err
			}
			if err := h.contentRepo.Update(txCtx, c); err != nil {
				return err
			}
			events = append(events, c.PullEvents()...)
		}

		if err := fresh.MarkFailed(StaleFailReason); err != nil {
			return err
		}
		if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
			return err
		}
		events = append(events, fresh.PullEvents()...)
		return h.outbox.StoreEvents(txCtx, events)
	})
}

func (h *RecoverStaleProcessingMediasHandler) requeueMedia(
	ctx context.Context,
	mediaID uuid.UUID,
	staleBefore, now time.Time,
) (bool, error) {
	var didWork bool
	err := h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		fresh, err := h.mediaRepo.GetByID(txCtx, mediaID)
		if err != nil {
			return err
		}
		if fresh == nil || fresh.Status != domainmedia.StatusProcessing {
			return nil
		}

		contents, err := h.contentRepo.ListByMediaID(txCtx, fresh.ID)
		if err != nil {
			return err
		}

		events := make([]event.DomainEvent, 0)

		if len(contents) == 0 {
			// Image without contents cannot recover via media.uploaded — fail it.
			if fresh.MediaType != domainmedia.MediaTypeVideo {
				if err := fresh.MarkFailed(StaleFailReason); err != nil {
					return err
				}
				if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
					return err
				}
				events = append(events, fresh.PullEvents()...)
				didWork = true
				return h.outbox.StoreEvents(txCtx, events)
			}
			if err := fresh.RequeueUploaded(); err != nil {
				return err
			}
			if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
				return err
			}
			events = append(events, fresh.PullEvents()...)
			didWork = true
			return h.outbox.StoreEvents(txCtx, events)
		}

		allTerminal := true
		for i := range contents {
			c := &contents[i]
			if c.IsTerminal() {
				continue
			}
			allTerminal = false
			// Only re-dispatch contents that never left "uploaded".
			// Actively analyzing contents are left alone until media max-age fails them.
			if c.Status != domaincontent.StatusUploaded || !c.UpdatedAt.Before(staleBefore) {
				continue
			}
			c.RequeueAnalysis()
			if err := h.contentRepo.Update(txCtx, c); err != nil {
				return err
			}
			events = append(events, c.PullEvents()...)
			didWork = true
		}

		if allTerminal {
			verdict := contentcmd.AggregateMediaVerdict(contents)
			if err := fresh.RenderGlobalVerdict(verdict); err != nil {
				return err
			}
			if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
				return err
			}
			events = append(events, fresh.PullEvents()...)
			didWork = true
			return h.outbox.StoreEvents(txCtx, events)
		}

		if !didWork {
			return nil
		}
		fresh.Touch(now)
		if err := h.mediaRepo.Update(txCtx, fresh); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, events)
	})
	return didWork, err
}
