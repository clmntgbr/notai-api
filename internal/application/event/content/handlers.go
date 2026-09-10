package content

import (
	"context"
	"encoding/json"
	"log"

	"go-api/internal/application/messaging"
	domaincontent "go-api/internal/domain/content"
)

type ContentCreatedHandler struct{}

func NewContentCreatedHandler() *ContentCreatedHandler { return &ContentCreatedHandler{} }

func (h *ContentCreatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincontent.ContentCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s contentId=%s campaignId=%s",
		domaincontent.EventTypeContentCreated,
		evt.ID,
		evt.ContentID,
		evt.CampaignID,
	)
	return nil
}

type ContentUploadedHandler struct{}

func NewContentUploadedHandler() *ContentUploadedHandler { return &ContentUploadedHandler{} }

func (h *ContentUploadedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincontent.ContentUploaded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s contentId=%s size=%d",
		domaincontent.EventTypeContentUploaded,
		evt.ID,
		evt.ContentID,
		evt.SizeBytes,
	)
	return nil
}

type ContentStatusChangedHandler struct{}

func NewContentStatusChangedHandler() *ContentStatusChangedHandler {
	return &ContentStatusChangedHandler{}
}

func (h *ContentStatusChangedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincontent.ContentStatusChanged
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s contentId=%s status=%s",
		domaincontent.EventTypeContentStatusChanged,
		evt.ID,
		evt.ContentID,
		evt.Status,
	)
	return nil
}
