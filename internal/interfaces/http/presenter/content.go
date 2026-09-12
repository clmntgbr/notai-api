package presenter

import (
	"fmt"
	"strconv"
	"time"

	contentcmd "go-api/internal/application/command/content"
	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type ContentDetailResponse struct {
	ID           string                   `json:"id"`
	CampaignID   string                   `json:"campaignId"`
	ClientID     string                   `json:"clientId"`
	Filename     string                   `json:"filename"`
	ContentType  string                   `json:"contentType"`
	Status       string                   `json:"status"`
	Label        string                   `json:"label,omitempty"`
	SizeBytes    *int64                   `json:"sizeBytes,omitempty"`
	ThumbnailURL string                   `json:"thumbnailUrl,omitempty"`
	Campaign     *ContentCampaignResponse `json:"campaign,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
}

type ContentCampaignResponse struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	BackgroundStatus       string     `json:"backgroundStatus"`
	BackgroundThumbnailURL string     `json:"backgroundThumbnailUrl,omitempty"`
	StartAt                *time.Time `json:"startAt,omitempty"`
	EndAt                  *time.Time `json:"endAt,omitempty"`
}

func NewContentDetailResponseFromView(view domaincontent.ContentView) ContentDetailResponse {
	label := ""
	if view.Label != nil {
		label = string(*view.Label)
	}
	return ContentDetailResponse{
		ID:           view.ID.String(),
		CampaignID:   view.CampaignID.String(),
		ClientID:     view.ClientID.String(),
		Filename:     view.Filename,
		ContentType:  view.ContentType,
		Status:       string(view.Status),
		Label:        label,
		SizeBytes:    view.SizeBytes,
		ThumbnailURL: contentThumbnailURL(view),
		CreatedAt:    view.CreatedAt,
		UpdatedAt:    view.UpdatedAt,
	}
}

func NewContentListResponseFromViews(
	views []domaincontent.ContentView,
	campaigns map[uuid.UUID]*domaincampaign.CampaignView,
) []ContentDetailResponse {
	items := make([]ContentDetailResponse, 0, len(views))
	for _, view := range views {
		item := NewContentDetailResponseFromView(view)
		if campaigns != nil {
			item.Campaign = contentCampaignResponse(campaigns[view.CampaignID])
		}
		items = append(items, item)
	}
	return items
}

func contentCampaignResponse(campaign *domaincampaign.CampaignView) *ContentCampaignResponse {
	if campaign == nil || campaign.IsDefault {
		return nil
	}
	return &ContentCampaignResponse{
		ID:               campaign.ID.String(),
		Name:             campaign.Name,
		BackgroundStatus: backgroundStatusOrNone(campaign.BackgroundStatus),
		BackgroundThumbnailURL: backgroundThumbnailURL(
			campaign.ID.String(),
			campaign.BackgroundStatus,
			campaign.BackgroundThumbnailKey,
			campaign.UpdatedAt,
		),
		StartAt: campaign.StartAt,
		EndAt:   campaign.EndAt,
	}
}

type ContentStatsResponse struct {
	PendingUpload int64 `json:"pendingUpload"`
	Uploaded      int64 `json:"uploaded"`
	Analyzing     int64 `json:"analyzing"`
	Failed        int64 `json:"failed"`
	Human         int64 `json:"human"`
	AIGenerated   int64 `json:"aiGenerated"`
	Uncertain     int64 `json:"uncertain"`
}

func NewContentStatsResponse(stats *domaincontent.ContentStats) ContentStatsResponse {
	if stats == nil {
		return ContentStatsResponse{}
	}
	return ContentStatsResponse{
		PendingUpload: stats.PendingUpload,
		Uploaded:      stats.Uploaded,
		Analyzing:     stats.Analyzing,
		Failed:        stats.Failed,
		Human:         stats.Human,
		AIGenerated:   stats.AIGenerated,
		Uncertain:     stats.Uncertain,
	}
}

func NewPresignContentsResponse(result *contentcmd.PresignContentsResult) PresignContentsResponse {
	items := make([]PresignContentItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, PresignContentItemResponse{
			ContentID: item.ContentID.String(),
			URL:       item.URL,
			ObjectKey: item.ObjectKey,
			Filename:  item.Filename,
		})
	}
	return PresignContentsResponse{
		CampaignID: result.CampaignID.String(),
		Items:      items,
	}
}

type PresignContentsResponse struct {
	CampaignID string                       `json:"campaignId"`
	Items      []PresignContentItemResponse `json:"items"`
}

type PresignContentItemResponse struct {
	ContentID string `json:"contentId"`
	URL       string `json:"url"`
	ObjectKey string `json:"objectKey"`
	Filename  string `json:"filename"`
}

func contentThumbnailURL(view domaincontent.ContentView) string {
	if view.ThumbnailKey == nil || *view.ThumbnailKey == "" {
		return ""
	}
	if view.Status == domaincontent.StatusPendingUpload || view.Status == domaincontent.StatusFailed {
		return ""
	}
	return fmt.Sprintf(
		"/api/contents/%s/thumbnail?v=%s",
		view.ID.String(),
		strconv.FormatInt(view.UpdatedAt.Unix(), 10),
	)
}
