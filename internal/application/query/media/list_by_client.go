package media

import (
	"context"
	"errors"
	"fmt"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListByClientQuery struct {
	ClientID    uuid.UUID
	CampaignIDs []uuid.UUID // optional; empty = all campaigns for the client
	Statuses    []string    // optional media statuses
	Verdicts    []string    // optional verdict labels (verdict->>'label')
	// From/To optional inclusive created_at window; both nil = no date filter.
	From  *time.Time
	To    *time.Time
	Query paginate.PaginateQuery
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

	if len(q.CampaignIDs) > 0 {
		campaigns, err := h.campaignRepo.FindByIDs(ctx, q.CampaignIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get campaigns: %w", err)
		}
		byID := make(map[uuid.UUID]domaincampaign.CampaignView, len(campaigns))
		for i := range campaigns {
			byID[campaigns[i].ID] = campaigns[i]
		}
		for _, id := range q.CampaignIDs {
			campaign, ok := byID[id]
			if !ok || campaign.ClientID != q.ClientID {
				return nil, errors.New("campaign not found")
			}
		}
	}

	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return nil, errors.New("from must be before to")
	}

	views, total, err := h.mediaRepo.FindPageByClientID(
		ctx,
		q.ClientID,
		q.CampaignIDs,
		q.Query,
		q.Statuses,
		q.Verdicts,
		q.From,
		q.To,
	)
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
