package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	campaigncmd "go-api/internal/application/command/campaign"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
)

type MediaUploadWebhookHandler struct {
	mediaBucket    string
	processHandler mediaUploadProcessHandler
}

func NewMediaUploadWebhookHandler(
	mediaBucket string,
	processHandler mediaUploadProcessHandler,
) *MediaUploadWebhookHandler {
	return &MediaUploadWebhookHandler{
		mediaBucket:    mediaBucket,
		processHandler: processHandler,
	}
}

func (h *MediaUploadWebhookHandler) ObjectCreated(c fiber.Ctx) error {
	payload := c.Body()

	var event dto.ObjectCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("MinIO webhook: invalid payload: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid payload",
		})
	}
	if err := validation.Struct(c, &event); err != nil {
		return err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		h.process(ctx, event)
	}()

	return c.SendStatus(fiber.StatusOK)
}

func (h *MediaUploadWebhookHandler) process(ctx context.Context, event dto.ObjectCreatedEvent) {
	for _, record := range event.Records {
		if record.S3.Bucket.Name != h.mediaBucket {
			continue
		}

		decodedKey, err := domaincampaign.DecodeObjectKey(record.S3.Object.Key)
		if err != nil {
			log.Printf("MinIO webhook: invalid object key %q: %v", record.S3.Object.Key, err)
			continue
		}
		if domaincampaign.IsThumbnailObjectKey(decodedKey) {
			continue
		}

		err = h.processHandler.Handle(ctx, campaigncmd.ProcessBackgroundUploadCommand{
			ObjectKey:   decodedKey,
			ContentType: record.S3.Object.ContentType,
			Size:        record.S3.Object.Size,
		})
		if err != nil {
			log.Printf("MinIO webhook: process failed key=%q: %v", decodedKey, err)
		}
	}
}
