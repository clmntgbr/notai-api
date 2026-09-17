package activity

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Insert(ctx context.Context, event *Event) (inserted bool, err error)
}

type ReadRepository interface {
	FindPageByClientID(
		ctx context.Context,
		clientID uuid.UUID,
		campaignIDs []uuid.UUID,
		from, to *time.Time,
		query paginate.PaginateQuery,
	) ([]View, int64, error)
}
