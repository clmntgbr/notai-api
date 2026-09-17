package activity

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainactivity "go-api/internal/domain/activity"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListByClientQuery struct {
	ClientID    uuid.UUID
	CampaignIDs []uuid.UUID // optional; empty = all campaigns
	// From/To optional inclusive occurred_at window; nil = no bound.
	From  *time.Time
	To    *time.Time
	Query paginate.PaginateQuery
}

type ListByClientHandler struct {
	repo         domainactivity.ReadRepository
	campaignRepo domaincampaign.CampaignReadRepository
}

func NewListByClientHandler(
	repo domainactivity.ReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
) *ListByClientHandler {
	return &ListByClientHandler{repo: repo, campaignRepo: campaignRepo}
}

func (h *ListByClientHandler) Handle(
	ctx context.Context,
	q ListByClientQuery,
) ([]domainactivity.View, int64, error) {
	if q.ClientID == uuid.Nil {
		return nil, 0, errors.New("clientId is required")
	}
	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return nil, 0, errors.New("from must be before to")
	}

	if len(q.CampaignIDs) > 0 {
		campaigns, err := h.campaignRepo.FindByIDs(ctx, q.CampaignIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get campaigns: %w", err)
		}
		byID := make(map[uuid.UUID]domaincampaign.CampaignView, len(campaigns))
		for i := range campaigns {
			byID[campaigns[i].ID] = campaigns[i]
		}
		for _, id := range q.CampaignIDs {
			campaign, ok := byID[id]
			if !ok || campaign.ClientID != q.ClientID {
				return nil, 0, errors.New("campaign not found")
			}
		}
	}

	views, total, err := h.repo.FindPageByClientID(
		ctx,
		q.ClientID,
		q.CampaignIDs,
		q.From,
		q.To,
		q.Query,
	)
	if err != nil {
		return nil, 0, errors.New("failed to list activity")
	}
	return views, total, nil
}
