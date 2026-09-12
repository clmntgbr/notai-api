package content

import (
	"context"

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
