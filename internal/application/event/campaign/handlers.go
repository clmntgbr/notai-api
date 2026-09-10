package campaign

import (
	"context"
	"encoding/json"
	"log"

	"go-api/internal/application/messaging"
	domaincampaign "go-api/internal/domain/campaign"
)

type CampaignCreatedHandler struct{}

func NewCampaignCreatedHandler() *CampaignCreatedHandler { return &CampaignCreatedHandler{} }

func (h *CampaignCreatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s campaignId=%s clientId=%s name=%s",
		domaincampaign.EventTypeCampaignCreated,
		evt.ID,
		evt.CampaignID,
		evt.ClientID,
		evt.Name,
	)
	return nil
}

type CampaignUpdatedHandler struct{}

func NewCampaignUpdatedHandler() *CampaignUpdatedHandler { return &CampaignUpdatedHandler{} }

func (h *CampaignUpdatedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s campaignId=%s name=%s",
		domaincampaign.EventTypeCampaignUpdated,
		evt.ID,
		evt.CampaignID,
		evt.Name,
	)
	return nil
}

type CampaignDeletedHandler struct{}

func NewCampaignDeletedHandler() *CampaignDeletedHandler { return &CampaignDeletedHandler{} }

func (h *CampaignDeletedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignDeleted
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	log.Printf(
		"event handled %s eventId=%s campaignId=%s clientId=%s",
		domaincampaign.EventTypeCampaignDeleted,
		evt.ID,
		evt.CampaignID,
		evt.ClientID,
	)
	return nil
}
