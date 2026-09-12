package media

import (
	"context"
	"errors"
	"fmt"

	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListByClientQuery struct {
	ClientID   uuid.UUID
	CampaignID uuid.UUID // optional; uuid.Nil = all campaigns for the client
	Query      paginate.PaginateQuery
}

type ListByClientResult struct {
	Views     []domainmedia.MediaView
	Total     int64
	Campaigns map[uuid.UUID]*domaincampaign.CampaignView
}

type ListByClientHandler struct {
	mediaRepo    domainmedia.MediaReadRepository
	campaignRepo domaincampaign.CampaignReadRepository
}

func NewListByClientHandler(
	mediaRepo domainmedia.MediaReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
) *ListByClientHandler {
	return &ListByClientHandler{mediaRepo: mediaRepo, campaignRepo: campaignRepo}
}

func (h *ListByClientHandler) Handle(
	ctx context.Context,
	q ListByClientQuery,
) (*ListByClientResult, error) {
	if q.ClientID == uuid.Nil {
		return nil, errors.New("clientId is required")
	}

	if q.CampaignID != uuid.Nil {
		campaign, err := h.campaignRepo.FindByID(ctx, q.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("failed to get campaign: %w", err)
		}
		if campaign == nil || campaign.ClientID != q.ClientID {
			return nil, errors.New("campaign not found")
		}
	}

	views, total, err := h.mediaRepo.FindPageByClientID(ctx, q.ClientID, q.CampaignID, q.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to list media: %w", err)
	}

	campaigns, err := h.loadCampaigns(ctx, views)
	if err != nil {
		return nil, err
	}

	return &ListByClientResult{
		Views:     views,
		Total:     total,
		Campaigns: campaigns,
	}, nil
}

func (h *ListByClientHandler) loadCampaigns(
	ctx context.Context,
	views []domainmedia.MediaView,
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
		return nil, fmt.Errorf("failed to get campaigns: %w", err)
	}

	campaigns := make(map[uuid.UUID]*domaincampaign.CampaignView, len(loaded))
	for i := range loaded {
		campaigns[loaded[i].ID] = &loaded[i]
	}
	return campaigns, nil
}
