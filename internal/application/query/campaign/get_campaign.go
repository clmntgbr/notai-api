package campaign

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"

	"github.com/google/uuid"
)

type GetCampaignByIDQuery struct {
	ID uuid.UUID
}

type GetCampaignByIDHandler struct {
	readRepo domaincampaign.CampaignReadRepository
}

func NewGetCampaignByIDHandler(
	readRepo domaincampaign.CampaignReadRepository,
) *GetCampaignByIDHandler {
	return &GetCampaignByIDHandler{readRepo: readRepo}
}

func (h *GetCampaignByIDHandler) Handle(
	ctx context.Context,
	q GetCampaignByIDQuery,
) (*domaincampaign.CampaignView, error) {
	view, err := h.readRepo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, errors.New("failed to get campaign")
	}
	if view == nil {
		return nil, errors.New("campaign not found")
	}
	return view, nil
}
