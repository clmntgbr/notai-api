package media

import (
	"context"
	"encoding/json"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type ExtractOnUploadedHandler struct {
	extract *mediacmd.ExtractFramesHandler
}

func NewExtractOnUploadedHandler(extract *mediacmd.ExtractFramesHandler) *ExtractOnUploadedHandler {
	return &ExtractOnUploadedHandler{extract: extract}
}

func (h *ExtractOnUploadedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaType != string(domainmedia.MediaTypeVideo) {
		return nil
	}
	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	if err := h.extract.Handle(ctx, mediacmd.ExtractFramesCommand{MediaID: mediaID}); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}
