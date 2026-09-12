package content

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ContentWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, content *Content) error
	Update(ctx context.Context, content *Content) error
	GetByID(ctx context.Context, id uuid.UUID) (*Content, error)
	GetByObjectKey(ctx context.Context, objectKey string) (*Content, error)
	ListByMediaID(ctx context.Context, mediaID uuid.UUID) ([]Content, error)
}

type ContentReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ContentView, error)
	FindPageByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		campaignID uuid.UUID,
		query paginate.PaginateQuery,
	) ([]ContentView, int64, error)
	CountStatsByClientID(ctx context.Context, clientID uuid.UUID) (*ContentStats, error)
}

type ContentStats struct {
	PendingUpload   int64
	Uploaded        int64
	Analyzing       int64
	Analyzed        int64
	Failed          int64
	Human           int64
	AIGenerated     int64
	Uncertain       int64
	MonthlyControls []ContentMonthlyStats
}

type ContentMonthlyStats struct {
	Month         string // YYYY-MM (UTC)
	PendingUpload int64
	Uploaded      int64
	Analyzing     int64
	Analyzed      int64
	Failed        int64
	Human         int64
	AIGenerated   int64
	Uncertain     int64
}

type ContentView struct {
	ID           uuid.UUID
	MediaID      uuid.UUID
	CampaignID   uuid.UUID
	ClientID     uuid.UUID
	Filename     string
	ContentType  string
	MediaType    string
	FrameIndex   *int
	TimestampMs  *int64
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Status       Status
	Label        *Label
	Confidence   *float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
