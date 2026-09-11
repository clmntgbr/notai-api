package content

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListContentsByCampaignQuery struct {
	CampaignID uuid.UUID
	ClientID   uuid.UUID
	Query      paginate.PaginateQuery
}

type ListContentsByCampaignHandler struct {
	contentRepo  domaincontent.ContentReadRepository
	campaignRepo domaincampaign.CampaignReadRepository
}

func NewListContentsByCampaignHandler(
	contentRepo domaincontent.ContentReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
) *ListContentsByCampaignHandler {
	return &ListContentsByCampaignHandler{
		contentRepo:  contentRepo,
		campaignRepo: campaignRepo,
	}
}

func (h *ListContentsByCampaignHandler) Handle(
	ctx context.Context,
	q ListContentsByCampaignQuery,
) ([]domaincontent.ContentView, int64, error) {
	if q.ClientID == uuid.Nil {
		return nil, 0, errors.New("clientId is required")
	}

	campaign, err := h.resolveCampaign(ctx, q.CampaignID, q.ClientID)
	if err != nil {
		return nil, 0, err
	}

	views, total, err := h.contentRepo.FindPageByCampaignID(ctx, campaign.ID, q.Query)
	if err != nil {
		return nil, 0, errors.New("failed to list contents")
	}
	return views, total, nil
}

func (h *ListContentsByCampaignHandler) resolveCampaign(
	ctx context.Context,
	campaignID, clientID uuid.UUID,
) (*domaincampaign.CampaignView, error) {
	var (
		campaign *domaincampaign.CampaignView
		err      error
	)
	if campaignID == uuid.Nil {
		campaign, err = h.campaignRepo.FindDefaultByClientID(ctx, clientID)
	} else {
		campaign, err = h.campaignRepo.FindByID(ctx, campaignID)
	}
	if err != nil {
		return nil, errors.New("failed to get campaign")
	}
	if campaign == nil || campaign.ClientID != clientID {
		return nil, errors.New("campaign not found")
	}
	return campaign, nil
}
