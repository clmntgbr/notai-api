package presenter

import (
	"fmt"
	"strconv"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
)

type CampaignDetailResponse struct {
	ID                     string    `json:"id"`
	ClientID               string    `json:"clientId"`
	Name                   string    `json:"name"`
	BackgroundStatus       string    `json:"backgroundStatus"`
	BackgroundThumbnailURL string    `json:"backgroundThumbnailUrl,omitempty"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

func NewCampaignDetailResponseFromView(view domaincampaign.CampaignView) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:               view.ID.String(),
		ClientID:         view.ClientID.String(),
		Name:             view.Name,
		BackgroundStatus: backgroundStatusOrNone(view.BackgroundStatus),
		BackgroundThumbnailURL: backgroundThumbnailURL(
			view.ID.String(),
			view.BackgroundStatus,
			view.BackgroundThumbnailKey,
			view.UpdatedAt,
		),
		CreatedAt: view.CreatedAt,
		UpdatedAt: view.UpdatedAt,
	}
}

func NewCampaignDetailResponseFromEntity(campaign domaincampaign.Campaign) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:               campaign.ID.String(),
		ClientID:         campaign.ClientID.String(),
		Name:             campaign.Name,
		BackgroundStatus: backgroundStatusOrNone(campaign.BackgroundStatus),
		BackgroundThumbnailURL: backgroundThumbnailURL(
			campaign.ID.String(),
			campaign.BackgroundStatus,
			campaign.BackgroundThumbnailKey,
			campaign.UpdatedAt,
		),
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

func backgroundStatusOrNone(status string) string {
	if status == "" {
		return domaincampaign.BackgroundStatusNone
	}
	return status
}

func backgroundThumbnailURL(campaignID, status, thumbnailKey string, updatedAt time.Time) string {
	if thumbnailKey == "" {
		return ""
	}
	if status != domaincampaign.BackgroundStatusReady && status != domaincampaign.BackgroundStatusPending {
		return ""
	}
	return fmt.Sprintf(
		"/api/campaigns/%s/thumbnail?v=%s",
		campaignID,
		strconv.FormatInt(updatedAt.Unix(), 10),
	)
}
