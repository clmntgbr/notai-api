package mediaupload

import (
	"context"
	"strings"

	campaigncmd "go-api/internal/application/command/campaign"
	mediacmd "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
)

// ObjectCreatedDispatcher routes MinIO object-created events to the right processor.
type ObjectCreatedDispatcher struct {
	background *campaigncmd.ProcessBackgroundUploadHandler
	media      *mediacmd.ProcessUploadHandler
}

func NewObjectCreatedDispatcher(
	background *campaigncmd.ProcessBackgroundUploadHandler,
	media *mediacmd.ProcessUploadHandler,
) *ObjectCreatedDispatcher {
	return &ObjectCreatedDispatcher{background: background, media: media}
}

func (d *ObjectCreatedDispatcher) Handle(
	ctx context.Context,
	objectKey, contentType string,
	size int64,
) error {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return nil
	}
	if domainmedia.IsThumbnailObjectKey(key) || domainmedia.IsFrameObjectKey(key) {
		return nil
	}
	if domainmedia.IsMediaObjectKey(key) {
		return d.media.Handle(ctx, mediacmd.ProcessUploadCommand{
			ObjectKey:   key,
			ContentType: contentType,
			Size:        size,
		})
	}
	return d.background.Handle(ctx, campaigncmd.ProcessBackgroundUploadCommand{
		ObjectKey:   key,
		ContentType: contentType,
		Size:        size,
	})
}
