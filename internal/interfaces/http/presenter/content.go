package presenter

import (
	"fmt"
	"strconv"
	"time"

	contentcmd "go-api/internal/application/command/content"
	domaincontent "go-api/internal/domain/content"
)

type ContentDetailResponse struct {
	ID           string    `json:"id"`
	CampaignID   string    `json:"campaignId"`
	ClientID     string    `json:"clientId"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"contentType"`
	Status       string    `json:"status"`
	SizeBytes    *int64    `json:"sizeBytes,omitempty"`
	ThumbnailURL string    `json:"thumbnailUrl,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewContentDetailResponseFromView(view domaincontent.ContentView) ContentDetailResponse {
	return ContentDetailResponse{
		ID:           view.ID.String(),
		CampaignID:   view.CampaignID.String(),
		ClientID:     view.ClientID.String(),
		Filename:     view.Filename,
		ContentType:  view.ContentType,
		Status:       string(view.Status),
		SizeBytes:    view.SizeBytes,
		ThumbnailURL: contentThumbnailURL(view),
		CreatedAt:    view.CreatedAt,
		UpdatedAt:    view.UpdatedAt,
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
