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

type ListContentsByCampaignResult struct {
	Views     []domaincontent.ContentView
	Total     int64
	Campaigns map[uuid.UUID]*domaincampaign.CampaignView
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
) (*ListContentsByCampaignResult, error) {
	if q.ClientID == uuid.Nil {
		return nil, errors.New("clientId is required")
	}

	if q.CampaignID != uuid.Nil {
		campaign, err := h.campaignRepo.FindByID(ctx, q.CampaignID)
		if err != nil {
			return nil, errors.New("failed to get campaign")
		}
		if campaign == nil || campaign.ClientID != q.ClientID {
			return nil, errors.New("campaign not found")
		}
	}

	views, total, err := h.contentRepo.FindPageByClientID(ctx, q.ClientID, q.CampaignID, q.Query)
	if err != nil {
		return nil, errors.New("failed to list contents")
	}

	campaigns, err := h.loadCampaigns(ctx, views)
	if err != nil {
		return nil, err
	}

	return &ListContentsByCampaignResult{
		Views:     views,
		Total:     total,
		Campaigns: campaigns,
	}, nil
}

func (h *ListContentsByCampaignHandler) loadCampaigns(
	ctx context.Context,
	views []domaincontent.ContentView,
) (map[uuid.UUID]*domaincampaign.CampaignView, error) {
	if len(views) == 0 {
		return map[uuid.UUID]*domaincampaign.CampaignView{}, nil
	}

	seen := make(map[uuid.UUID]struct{}, len(views))
	ids := make([]uuid.UUID, 0, len(views))
	for _, view := range views {
		if _, ok := seen[view.CampaignID]; ok {
			continue
		}
		seen[view.CampaignID] = struct{}{}
		ids = append(ids, view.CampaignID)
	}

	loaded, err := h.campaignRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, errors.New("failed to get campaigns")
	}

	campaigns := make(map[uuid.UUID]*domaincampaign.CampaignView, len(loaded))
	for i := range loaded {
		campaign := loaded[i]
		campaigns[campaign.ID] = &campaign
	}
	return campaigns, nil
}
