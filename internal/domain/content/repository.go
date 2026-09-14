package content

import (
	"context"
	"time"

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
	CountByClientIDAndStatus(ctx context.Context, clientID uuid.UUID, status string) (int64, error)
	CountByClientIDAndStatusInPeriod(
		ctx context.Context,
		clientID uuid.UUID,
		status string,
		from, to time.Time,
	) (int64, error)
	CountQuotaUnitsByClientIDInPeriod(
		ctx context.Context,
		clientID uuid.UUID,
		from, to time.Time,
	) (int64, error)
	CountByWorkspaceIDAndStatus(ctx context.Context, workspaceID uuid.UUID, status string) (int64, error)
	CountQuotaUnitsByWorkspaceIDInPeriod(
		ctx context.Context,
		workspaceID uuid.UUID,
		from, to time.Time,
	) (int64, error)
}
