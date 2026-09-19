package campaign

import (
	"context"
	"encoding/json"
	"strings"

	"go-api/internal/application/messaging"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/port"
)

// DeleteObjectsOnDeletedHandler purges campaign background objects from S3/MinIO.
type DeleteObjectsOnDeletedHandler struct {
	storage port.Storage
}

func NewDeleteObjectsOnDeletedHandler(storage port.Storage) *DeleteObjectsOnDeletedHandler {
	return &DeleteObjectsOnDeletedHandler{storage: storage}
}

func (h *DeleteObjectsOnDeletedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	if key := strings.TrimSpace(evt.BackgroundPendingKey); key != "" {
		if err := h.storage.Delete(ctx, key); err != nil {
			return messaging.Retryable(err)
		}
	}
	if key := strings.TrimSpace(evt.BackgroundThumbnailKey); key != "" {
		if err := h.storage.DeleteThumbnail(ctx, key); err != nil {
			return messaging.Retryable(err)
		}
	}
	return nil
}
