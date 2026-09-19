package media

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type MediaWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, media *Media) error
	Update(ctx context.Context, media *Media) error
	GetByID(ctx context.Context, id uuid.UUID) (*Media, error)
	GetByObjectKey(ctx context.Context, objectKey string) (*Media, error)
	ListProcessingUpdatedBefore(ctx context.Context, before time.Time, limit int) ([]*Media, error)
	ListActiveByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*Media, error)
}

type MediaReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*MediaView, error)
	FindPageByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		campaignIDs []uuid.UUID,
		query paginate.PaginateQuery,
		statuses, verdicts []string,
		from, to *time.Time,
	) ([]MediaView, int64, error)
	FindContentsByMediaID(ctx context.Context, mediaID uuid.UUID) ([]ContentChildView, error)
	CountStatsByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		campaignID *uuid.UUID,
		from, to time.Time,
	) (*MediaStats, error)
	// SumStorageBytesByWorkspaceID sums media originals + extracted frame object sizes
	// (frame_index IS NOT NULL) for the workspace billing scope.
	SumStorageBytesByWorkspaceID(ctx context.Context, workspaceID uuid.UUID) (int64, error)
}

type MediaStats struct {
	PendingUpload int64
	Uploaded      int64
	Processing    int64
	Analyzed      int64
	Failed        int64
	Human         int64
	AIGenerated   int64
	Uncertain     int64
	From          time.Time
	To            time.Time
	DailyControls []MediaDailyStats
	KPIs          MediaDashboardKPIs
}

// MediaDashboardKPIs are the agency home cards for the selected period vs the previous
// window of equal duration.
type MediaDashboardKPIs struct {
	Month string

	Verifications              int64
	VerificationsChangePercent *float64 // nil when previous period had 0
	PlanIncluded               *int64   // nil until billing plans exist

	AuthenticityRatePercent  float64
	AuthenticityChangePoints *float64 // nil when previous period had 0 verifications
	ValidatedCount           int64    // human-labeled in period

	ToReviewCount           int64
	ToReviewChangePercent   *float64
	AIGeneratedCount        int64
	AIGeneratedSharePercent float64 // share of period verifications
}

// MediaDailyStats is one UTC calendar day of status/verdict counts.
type MediaDailyStats struct {
	Day           string // YYYY-MM-DD (UTC)
	PendingUpload int64
	Uploaded      int64
	Processing    int64
	Analyzed      int64
	Failed        int64
	Human         int64
	AIGenerated   int64
	Uncertain     int64
}

type MediaView struct {
	ID            uuid.UUID
	CampaignID    uuid.UUID
	ClientID      uuid.UUID
	Filename      string
	ContentType   string
	MediaType     MediaType
	ObjectKey     string
	SizeBytes     *int64
	Status        Status
	Verdict       *Verdict
	FailureReason string
	AnalyzedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ContentChildView is a nested content row for media detail drill-down.
type ContentChildView struct {
	ID           uuid.UUID
	MediaID      uuid.UUID
	FrameIndex   *int
	TimestampMs  *int64
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Status       string
	Label        *string
	Confidence   *float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
