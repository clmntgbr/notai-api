package presenter

import (
	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type MediaVerdictResponse struct {
	Label        string `json:"label"`
	FlaggedCount int    `json:"flaggedCount"`
	TotalCount   int    `json:"totalCount"`
	FailedCount  int    `json:"failedCount"`
}

type MediaListItemResponse struct {
	ID           string                 `json:"id"`
	CampaignID   string                 `json:"campaignId"`
	Filename     string                 `json:"filename"`
	MediaType    string                 `json:"mediaType"`
	Status       string                 `json:"status"`
	Verdict      *MediaVerdictResponse  `json:"verdict,omitempty"`
	ThumbnailURL *string                `json:"thumbnailUrl,omitempty"`
	Campaign     *MediaCampaignResponse `json:"campaign,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	AnalyzedAt   *time.Time             `json:"analyzedAt,omitempty"`
}

type MediaCampaignResponse struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	BackgroundStatus       string     `json:"backgroundStatus"`
	BackgroundThumbnailURL string     `json:"backgroundThumbnailUrl,omitempty"`
	StartAt                *time.Time `json:"startAt,omitempty"`
	EndAt                  *time.Time `json:"endAt,omitempty"`
}

type MediaContentChildResponse struct {
	ID           string  `json:"id"`
	FrameIndex   *int    `json:"frameIndex,omitempty"`
	TimestampMs  *int64  `json:"timestampMs,omitempty"`
	Status       string  `json:"status"`
	ThumbnailURL *string `json:"thumbnailUrl,omitempty"`
	Verdict      *struct {
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
	CampaignID string                     `json:"campaignId"`
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

func NewMediaListResponseFromViews(
	views []domainmedia.MediaView,
	campaigns map[uuid.UUID]*domaincampaign.CampaignView,
) []MediaListItemResponse {
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
			url := "/api/medias/" + v.ID.String() + "/thumbnail?v=" + strconv.FormatInt(v.UpdatedAt.UnixNano(), 10)
			item.ThumbnailURL = &url
		}
		if campaigns != nil {
			item.Campaign = mediaCampaignResponse(campaigns[v.CampaignID])
		}
		out = append(out, item)
	}
	return out
}

func mediaCampaignResponse(campaign *domaincampaign.CampaignView) *MediaCampaignResponse {
	if campaign == nil || campaign.IsDefault {
		return nil
	}
	return &MediaCampaignResponse{
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
		if c.ThumbnailKey != nil && *c.ThumbnailKey != "" {
			url := "/api/medias/" + v.ID.String() + "/contents/" + c.ID.String() +
				"/thumbnail?v=" + strconv.FormatInt(c.UpdatedAt.UnixNano(), 10)
			child.ThumbnailURL = &url
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

type MediaStatsResponse struct {
	PendingUpload   int64                          `json:"pendingUpload"`
	Uploaded        int64                          `json:"uploaded"`
	Processing      int64                          `json:"processing"`
	Analyzed        int64                          `json:"analyzed"`
	Failed          int64                          `json:"failed"`
	Human           int64                          `json:"human"`
	AIGenerated     int64                          `json:"aiGenerated"`
	Uncertain       int64                          `json:"uncertain"`
	MonthlyControls []MediaMonthlyControlsResponse `json:"monthlyControls"`
	KPIs            MediaDashboardKPIsResponse     `json:"kpis"`
}

type MediaDashboardKPIsResponse struct {
	Month                      string   `json:"month"`
	Verifications              int64    `json:"verifications"`
	VerificationsChangePercent *float64 `json:"verificationsChangePercent"`
	PlanIncluded               *int64   `json:"planIncluded"`
	AuthenticityRatePercent    float64  `json:"authenticityRatePercent"`
	AuthenticityChangePoints   *float64 `json:"authenticityChangePoints"`
	ValidatedCount             int64    `json:"validatedCount"`
	ToReviewCount              int64    `json:"toReviewCount"`
	ToReviewChangePercent      *float64 `json:"toReviewChangePercent"`
	AIGeneratedCount           int64    `json:"aiGeneratedCount"`
	AIGeneratedSharePercent    float64  `json:"aiGeneratedSharePercent"`
}

type MediaMonthlyControlsResponse struct {
	Month         string `json:"month"`
	PendingUpload int64  `json:"pendingUpload"`
	Uploaded      int64  `json:"uploaded"`
	Processing    int64  `json:"processing"`
	Analyzed      int64  `json:"analyzed"`
	Failed        int64  `json:"failed"`
	Human         int64  `json:"human"`
	AIGenerated   int64  `json:"aiGenerated"`
	Uncertain     int64  `json:"uncertain"`
}

func NewMediaStatsResponse(stats *domainmedia.MediaStats) MediaStatsResponse {
	if stats == nil {
		return MediaStatsResponse{
			MonthlyControls: make([]MediaMonthlyControlsResponse, 0),
			KPIs:            MediaDashboardKPIsResponse{},
		}
	}
	monthly := make([]MediaMonthlyControlsResponse, 0, len(stats.MonthlyControls))
	for _, m := range stats.MonthlyControls {
		monthly = append(monthly, MediaMonthlyControlsResponse{
			Month:         m.Month,
			PendingUpload: m.PendingUpload,
			Uploaded:      m.Uploaded,
			Processing:    m.Processing,
			Analyzed:      m.Analyzed,
			Failed:        m.Failed,
			Human:         m.Human,
			AIGenerated:   m.AIGenerated,
			Uncertain:     m.Uncertain,
		})
	}
	return MediaStatsResponse{
		PendingUpload:   stats.PendingUpload,
		Uploaded:        stats.Uploaded,
		Processing:      stats.Processing,
		Analyzed:        stats.Analyzed,
		Failed:          stats.Failed,
		Human:           stats.Human,
		AIGenerated:     stats.AIGenerated,
		Uncertain:       stats.Uncertain,
		MonthlyControls: monthly,
		KPIs: MediaDashboardKPIsResponse{
			Month:                      stats.KPIs.Month,
			Verifications:              stats.KPIs.Verifications,
			VerificationsChangePercent: stats.KPIs.VerificationsChangePercent,
			PlanIncluded:               stats.KPIs.PlanIncluded,
			AuthenticityRatePercent:    stats.KPIs.AuthenticityRatePercent,
			AuthenticityChangePoints:   stats.KPIs.AuthenticityChangePoints,
			ValidatedCount:             stats.KPIs.ValidatedCount,
			ToReviewCount:              stats.KPIs.ToReviewCount,
			ToReviewChangePercent:      stats.KPIs.ToReviewChangePercent,
			AIGeneratedCount:           stats.KPIs.AIGeneratedCount,
			AIGeneratedSharePercent:    stats.KPIs.AIGeneratedSharePercent,
		},
	}
}
