package presenter

import (
	"time"

	domaincampaign "go-api/internal/domain/campaign"
)

type CampaignDetailResponse struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"clientId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewCampaignDetailResponseFromView(view domaincampaign.CampaignView) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:        view.ID.String(),
		ClientID:  view.ClientID.String(),
		Name:      view.Name,
		CreatedAt: view.CreatedAt,
		UpdatedAt: view.UpdatedAt,
	}
}

func NewCampaignDetailResponseFromEntity(campaign domaincampaign.Campaign) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:        campaign.ID.String(),
		ClientID:  campaign.ClientID.String(),
		Name:      campaign.Name,
		CreatedAt: campaign.CreatedAt,
		UpdatedAt: campaign.UpdatedAt,
	}
}

func NewCampaignListResponseFromViews(views []domaincampaign.CampaignView) []CampaignDetailResponse {
	items := make([]CampaignDetailResponse, 0, len(views))
	for _, view := range views {
		items = append(items, NewCampaignDetailResponseFromView(view))
	}
	return items
}
