package analysisresult

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Upsert(ctx context.Context, result *Result) error
	FindByContentID(ctx context.Context, contentID uuid.UUID) ([]Result, error)
	FindByContentIDs(ctx context.Context, contentIDs []uuid.UUID) (map[uuid.UUID][]Result, error)
}
