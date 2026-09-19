package media

import (
	"context"
	"encoding/json"
	"strings"

	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

// DeleteObjectsOnDeletedHandler purges S3/MinIO objects for a soft-deleted media.
type DeleteObjectsOnDeletedHandler struct {
	storage port.Storage
}

func NewDeleteObjectsOnDeletedHandler(storage port.Storage) *DeleteObjectsOnDeletedHandler {
	return &DeleteObjectsOnDeletedHandler{storage: storage}
}

func (h *DeleteObjectsOnDeletedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	objectKeys := map[string]struct{}{}
	thumbnailKeys := map[string]struct{}{}

	addKey := func(set map[string]struct{}, key string) {
		key = strings.TrimSpace(key)
		if key != "" {
			set[key] = struct{}{}
		}
	}

	addKey(objectKeys, evt.ObjectKey)
	addKey(thumbnailKeys, evt.ThumbnailKey)
	for _, content := range evt.Contents {
		addKey(objectKeys, content.ObjectKey)
		addKey(thumbnailKeys, content.ThumbnailKey)
	}

	for key := range objectKeys {
		if err := h.storage.Delete(ctx, key); err != nil {
			return messaging.Retryable(err)
		}
	}
	for key := range thumbnailKeys {
		if err := h.storage.DeleteThumbnail(ctx, key); err != nil {
			return messaging.Retryable(err)
		}
	}
	return nil
}
