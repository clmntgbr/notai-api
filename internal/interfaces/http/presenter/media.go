package presenter

import (
	querymedia "go-api/internal/application/query/media"
	mediacmd "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"strconv"
	"time"
)

type MediaVerdictResponse struct {
	Label        string `json:"label"`
	FlaggedCount int    `json:"flaggedCount"`
	TotalCount   int    `json:"totalCount"`
	FailedCount  int    `json:"failedCount"`
}

type MediaListItemResponse struct {
	ID          string                `json:"id"`
	CampaignID  string                `json:"campaignId"`
	Filename    string                `json:"filename"`
	MediaType   string                `json:"mediaType"`
	Status      string                `json:"status"`
	Verdict     *MediaVerdictResponse `json:"verdict,omitempty"`
	ThumbnailURL *string              `json:"thumbnailUrl,omitempty"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	AnalyzedAt  *time.Time            `json:"analyzedAt,omitempty"`
}

type MediaContentChildResponse struct {
	ID          string     `json:"id"`
	FrameIndex  *int       `json:"frameIndex,omitempty"`
	TimestampMs *int64     `json:"timestampMs,omitempty"`
	Status      string     `json:"status"`
	Verdict     *struct {
		Label      string  `json:"label"`
		Confidence float64 `json:"confidence"`
	} `json:"verdict,omitempty"`
}

type MediaDetailResponse struct {
	ID         string                      `json:"id"`
	CampaignID string                      `json:"campaignId"`
	Filename   string                      `json:"filename"`
	MediaType  string                      `json:"mediaType"`
	Status     string                      `json:"status"`
	Verdict    *MediaVerdictResponse       `json:"verdict,omitempty"`
	Contents   []MediaContentChildResponse `json:"contents"`
	CreatedAt  time.Time                   `json:"createdAt"`
	UpdatedAt  time.Time                   `json:"updatedAt"`
	AnalyzedAt *time.Time                  `json:"analyzedAt,omitempty"`
}

type PresignMediaResponse struct {
	CampaignID string                `json:"campaignId"`
	Items      []PresignMediaItemResponse `json:"items"`
}

type PresignMediaItemResponse struct {
	MediaID   string `json:"mediaId"`
	URL       string `json:"url"`
	ObjectKey string `json:"objectKey"`
	Filename  string `json:"filename"`
	MediaType string `json:"mediaType"`
}

func NewPresignMediaResponse(result *mediacmd.PresignMediaResult) PresignMediaResponse {
	items := make([]PresignMediaItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, PresignMediaItemResponse{
			MediaID:   item.MediaID.String(),
			URL:       item.URL,
			ObjectKey: item.ObjectKey,
			Filename:  item.Filename,
			MediaType: string(item.MediaType),
		})
	}
	return PresignMediaResponse{
		CampaignID: result.CampaignID.String(),
		Items:      items,
	}
}

func NewMediaListResponseFromViews(views []domainmedia.MediaView) []MediaListItemResponse {
	out := make([]MediaListItemResponse, 0, len(views))
	for _, v := range views {
		item := MediaListItemResponse{
			ID:         v.ID.String(),
			CampaignID: v.CampaignID.String(),
			Filename:   v.Filename,
			MediaType:  string(v.MediaType),
			Status:     string(v.Status),
			CreatedAt:  v.CreatedAt,
			UpdatedAt:  v.UpdatedAt,
			AnalyzedAt: v.AnalyzedAt,
		}
		if v.Verdict != nil {
			item.Verdict = &MediaVerdictResponse{
				Label:        string(v.Verdict.Label),
				FlaggedCount: v.Verdict.FlaggedCount,
				TotalCount:   v.Verdict.TotalCount,
				FailedCount:  v.Verdict.FailedCount,
			}
		}
		if v.MediaType == domainmedia.MediaTypeImage {
			url := "/api/media/" + v.ID.String() + "/thumbnail?v=" + strconv.FormatInt(v.UpdatedAt.UnixNano(), 10)
			item.ThumbnailURL = &url
		}
		out = append(out, item)
	}
	return out
}

func NewMediaDetailResponse(result *querymedia.GetByIDResult) MediaDetailResponse {
	v := result.Media
	resp := MediaDetailResponse{
		ID:         v.ID.String(),
		CampaignID: v.CampaignID.String(),
		Filename:   v.Filename,
		MediaType:  string(v.MediaType),
		Status:     string(v.Status),
		Contents:   make([]MediaContentChildResponse, 0, len(result.Contents)),
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
		AnalyzedAt: v.AnalyzedAt,
	}
	if v.Verdict != nil {
		resp.Verdict = &MediaVerdictResponse{
			Label:        string(v.Verdict.Label),
			FlaggedCount: v.Verdict.FlaggedCount,
			TotalCount:   v.Verdict.TotalCount,
			FailedCount:  v.Verdict.FailedCount,
		}
	}
	for _, c := range result.Contents {
		child := MediaContentChildResponse{
			ID:          c.ID.String(),
			FrameIndex:  c.FrameIndex,
			TimestampMs: c.TimestampMs,
			Status:      c.Status,
		}
		if c.Label != nil && *c.Label != "" {
			conf := 0.0
			if c.Confidence != nil {
				conf = *c.Confidence
			}
			child.Verdict = &struct {
				Label      string  `json:"label"`
				Confidence float64 `json:"confidence"`
			}{Label: *c.Label, Confidence: conf}
		}
		resp.Contents = append(resp.Contents, child)
	}
	return resp
}
