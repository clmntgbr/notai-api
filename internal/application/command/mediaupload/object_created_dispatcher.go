package mediaupload

import (
	"context"
	"strings"

	campaigncmd "go-api/internal/application/command/campaign"
	contentcmd "go-api/internal/application/command/content"
	domaincontent "go-api/internal/domain/content"
)

// ObjectCreatedDispatcher routes MinIO object-created events to the right processor.
type ObjectCreatedDispatcher struct {
	background *campaigncmd.ProcessBackgroundUploadHandler
	content    *contentcmd.ProcessUploadHandler
}

func NewObjectCreatedDispatcher(
	background *campaigncmd.ProcessBackgroundUploadHandler,
	content *contentcmd.ProcessUploadHandler,
) *ObjectCreatedDispatcher {
	return &ObjectCreatedDispatcher{background: background, content: content}
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
	if domaincontent.IsContentObjectKey(key) {
		return d.content.Handle(ctx, contentcmd.ProcessUploadCommand{
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
