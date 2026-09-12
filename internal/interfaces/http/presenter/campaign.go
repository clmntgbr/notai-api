package presenter

import (
	"fmt"
	"strconv"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
)

type CampaignContentCountsResponse struct {
	PendingUpload int64 `json:"pendingUpload"`
	Uploaded      int64 `json:"uploaded"`
	Analyzing     int64 `json:"analyzing"`
	Failed        int64 `json:"failed"`
	Human         int64 `json:"human"`
	AIGenerated   int64 `json:"aiGenerated"`
	Uncertain     int64 `json:"uncertain"`
}

type CampaignDetailResponse struct {
	ID                     string                        `json:"id"`
	ClientID               string                        `json:"clientId"`
	Name                   string                        `json:"name"`
	IsDefault              bool                          `json:"isDefault"`
	BackgroundStatus       string                        `json:"backgroundStatus"`
	BackgroundThumbnailURL string                        `json:"backgroundThumbnailUrl,omitempty"`
	StartAt                *time.Time                    `json:"startAt,omitempty"`
	EndAt                  *time.Time                    `json:"endAt,omitempty"`
	ContentCounts          CampaignContentCountsResponse `json:"contentCounts"`
	CreatedAt              time.Time                     `json:"createdAt"`
	UpdatedAt              time.Time                     `json:"updatedAt"`
}

func NewCampaignDetailResponseFromView(view domaincampaign.CampaignView) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:               view.ID.String(),
		ClientID:         view.ClientID.String(),
		Name:             view.Name,
		IsDefault:        view.IsDefault,
		BackgroundStatus: backgroundStatusOrNone(view.BackgroundStatus),
		BackgroundThumbnailURL: backgroundThumbnailURL(
			view.ID.String(),
			view.BackgroundStatus,
			view.BackgroundThumbnailKey,
			view.UpdatedAt,
		),
		StartAt: view.StartAt,
		EndAt:   view.EndAt,
		ContentCounts: CampaignContentCountsResponse{
			PendingUpload: view.ContentPendingUploadCount,
			Uploaded:      view.ContentUploadedCount,
			Analyzing:     view.ContentAnalyzingCount,
			Failed:        view.ContentFailedCount,
			Human:         view.ContentHumanCount,
			AIGenerated:   view.ContentAIGeneratedCount,
			Uncertain:     view.ContentUncertainCount,
		},
		CreatedAt: view.CreatedAt,
		UpdatedAt: view.UpdatedAt,
	}
}

func NewCampaignDetailResponseFromEntity(campaign domaincampaign.Campaign) CampaignDetailResponse {
	return CampaignDetailResponse{
		ID:               campaign.ID.String(),
		ClientID:         campaign.ClientID.String(),
		Name:             campaign.Name,
		IsDefault:        campaign.IsDefault,
		BackgroundStatus: backgroundStatusOrNone(campaign.BackgroundStatus),
		BackgroundThumbnailURL: backgroundThumbnailURL(
			campaign.ID.String(),
			campaign.BackgroundStatus,
			campaign.BackgroundThumbnailKey,
			campaign.UpdatedAt,
		),
		StartAt:       campaign.StartAt,
		EndAt:         campaign.EndAt,
		ContentCounts: CampaignContentCountsResponse{},
		CreatedAt:     campaign.CreatedAt,
		UpdatedAt:     campaign.UpdatedAt,
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
