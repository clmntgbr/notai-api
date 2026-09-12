package media

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListByCampaignQuery struct {
	ClientID   uuid.UUID
	CampaignID uuid.UUID
	Query      paginate.PaginateQuery
}

type ListByCampaignResult struct {
	Views []domainmedia.MediaView
	Total int64
}

type ListByCampaignHandler struct {
	mediaRepo    domainmedia.MediaReadRepository
	campaignRepo domaincampaign.CampaignReadRepository
}

func NewListByCampaignHandler(
	mediaRepo domainmedia.MediaReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
) *ListByCampaignHandler {
	return &ListByCampaignHandler{mediaRepo: mediaRepo, campaignRepo: campaignRepo}
}

func (h *ListByCampaignHandler) Handle(ctx context.Context, q ListByCampaignQuery) (*ListByCampaignResult, error) {
	if q.ClientID == uuid.Nil || q.CampaignID == uuid.Nil {
		return nil, errors.New("campaign not found")
	}
	campaign, err := h.campaignRepo.FindByID(ctx, q.CampaignID)
	if err != nil {
		return nil, err
	}
	if campaign == nil || campaign.ClientID != q.ClientID {
		return nil, errors.New("campaign not found")
	}

	views, total, err := h.mediaRepo.FindPageByCampaignID(ctx, q.ClientID, q.CampaignID, q.Query)
	if err != nil {
		return nil, err
	}
	return &ListByCampaignResult{Views: views, Total: total}, nil
}
