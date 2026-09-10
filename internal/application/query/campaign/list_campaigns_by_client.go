package campaign

import (
	"context"
	"errors"

	"go-api/internal/domain/paginate"
	domaincampaign "go-api/internal/domain/campaign"

	"github.com/google/uuid"
)

type ListCampaignsByClientQuery struct {
	ClientID uuid.UUID
	Query    paginate.PaginateQuery
}

type ListCampaignsByClientHandler struct {
	readRepo domaincampaign.CampaignReadRepository
}

func NewListCampaignsByClientHandler(
	readRepo domaincampaign.CampaignReadRepository,
) *ListCampaignsByClientHandler {
	return &ListCampaignsByClientHandler{readRepo: readRepo}
}

func (h *ListCampaignsByClientHandler) Handle(
	ctx context.Context,
	q ListCampaignsByClientQuery,
) ([]domaincampaign.CampaignView, int64, error) {
	if q.ClientID == uuid.Nil {
		return nil, 0, errors.New("clientId is required")
	}
	views, total, err := h.readRepo.FindPageByClientID(ctx, q.ClientID, q.Query)
	if err != nil {
		return nil, 0, errors.New("failed to list campaigns")
	}
	return views, total, nil
}
