package content

import (
	"context"
	"encoding/json"
	"log"

	contentcmd "go-api/internal/application/command/content"
	"go-api/internal/application/messaging"
	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type AnalyzeOnUploadedHandler struct {
	analyze *contentcmd.AnalyzeContentHandler
}

func NewAnalyzeOnUploadedHandler(analyze *contentcmd.AnalyzeContentHandler) *AnalyzeOnUploadedHandler {
	return &AnalyzeOnUploadedHandler{analyze: analyze}
}

func (h *AnalyzeOnUploadedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincontent.ContentUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	contentID, err := uuid.Parse(evt.ContentID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	log.Printf("content analysis started content_id=%s", contentID)
	if err := h.analyze.Handle(ctx, contentcmd.AnalyzeContentCommand{ContentID: contentID}); err != nil {
		return messaging.Retryable(err)
	}
	log.Printf("content analysis finished content_id=%s", contentID)
	return nil
}
