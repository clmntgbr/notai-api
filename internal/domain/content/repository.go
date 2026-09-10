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
}

type ContentReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ContentView, error)
	FindPageByCampaignID(
		ctx context.Context,
		campaignID uuid.UUID,
		query paginate.PaginateQuery,
	) ([]ContentView, int64, error)
}

type ContentView struct {
	ID           uuid.UUID
	CampaignID   uuid.UUID
	ClientID     uuid.UUID
	Filename     string
	ContentType  string
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
