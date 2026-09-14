package media

import (
	"context"
	"errors"

	"go-api/internal/domain/analysisresult"
	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetByIDQuery struct {
	ID       uuid.UUID
	ClientID uuid.UUID
}

type GetByIDResult struct {
	Media             domainmedia.MediaView
	Contents          []domainmedia.ContentChildView
	Campaign          *domaincampaign.CampaignView
	AnalysisByContent map[uuid.UUID][]analysisresult.Result
}

type GetByIDHandler struct {
	mediaRepo    domainmedia.MediaReadRepository
	campaignRepo domaincampaign.CampaignReadRepository
	analysisRepo analysisresult.WriteRepository
}

func NewGetByIDHandler(
	mediaRepo domainmedia.MediaReadRepository,
	campaignRepo domaincampaign.CampaignReadRepository,
	analysisRepo analysisresult.WriteRepository,
) *GetByIDHandler {
	return &GetByIDHandler{
		mediaRepo:    mediaRepo,
		campaignRepo: campaignRepo,
		analysisRepo: analysisRepo,
	}
}

func (h *GetByIDHandler) Handle(ctx context.Context, q GetByIDQuery) (*GetByIDResult, error) {
	if q.ID == uuid.Nil {
		return nil, errors.New("media not found")
	}
	view, err := h.mediaRepo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if view == nil || (q.ClientID != uuid.Nil && view.ClientID != q.ClientID) {
		return nil, errors.New("media not found")
	}

	contents, err := h.mediaRepo.FindContentsByMediaID(ctx, view.ID)
	if err != nil {
		return nil, err
	}

	var campaign *domaincampaign.CampaignView
	if h.campaignRepo != nil {
		campaign, err = h.campaignRepo.FindByID(ctx, view.CampaignID)
		if err != nil {
			return nil, err
		}
	}

	analysisByContent := map[uuid.UUID][]analysisresult.Result{}
	if h.analysisRepo != nil && len(contents) > 0 {
		ids := make([]uuid.UUID, 0, len(contents))
		for _, child := range contents {
			ids = append(ids, child.ID)
		}
		analysisByContent, err = h.analysisRepo.FindByContentIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	return &GetByIDResult{
		Media:             *view,
		Contents:          contents,
		Campaign:          campaign,
		AnalysisByContent: analysisByContent,
	}, nil
}
