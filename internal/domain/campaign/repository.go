package campaign

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type CampaignWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, campaign *Campaign) error
	Update(ctx context.Context, campaign *Campaign) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Campaign, error)
}

type CampaignReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*CampaignView, error)
	FindPageByClientID(ctx context.Context, clientID uuid.UUID, query paginate.PaginateQuery) ([]CampaignView, int64, error)
}

type CampaignView struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
