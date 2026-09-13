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
}

type MediaReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*MediaView, error)
	FindPageByClientID(
		ctx context.Context,
		clientID, campaignID uuid.UUID,
		query paginate.PaginateQuery,
	) ([]MediaView, int64, error)
	FindContentsByMediaID(ctx context.Context, mediaID uuid.UUID) ([]ContentChildView, error)
	CountStatsByClientID(ctx context.Context, clientID uuid.UUID) (*MediaStats, error)
}

type MediaStats struct {
	PendingUpload   int64
	Uploaded        int64
	Processing      int64
	Analyzed        int64
	Failed          int64
	Human           int64
	AIGenerated     int64
	Uncertain       int64
	MonthlyControls []MediaMonthlyStats
	KPIs            MediaDashboardKPIs
}

// MediaDashboardKPIs are the agency home cards (current UTC month vs previous).
type MediaDashboardKPIs struct {
	Month string

	Verifications              int64
	VerificationsChangePercent *float64 // nil when previous month had 0
	PlanIncluded               *int64   // nil until billing plans exist

	AuthenticityRatePercent  float64
	AuthenticityChangePoints *float64 // nil when previous month had 0 verifications
	ValidatedCount           int64    // human-labeled this month

	ToReviewCount           int64
	ToReviewChangePercent   *float64
	AIGeneratedCount        int64
	AIGeneratedSharePercent float64 // share of this month's verifications
}

type MediaMonthlyStats struct {
	Month         string // YYYY-MM (UTC)
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
	ID          uuid.UUID
	CampaignID  uuid.UUID
	ClientID    uuid.UUID
	Filename    string
	ContentType string
	MediaType   MediaType
	ObjectKey   string
	SizeBytes   *int64
	Status      Status
	Verdict     *Verdict
	AnalyzedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
