package client

import (
	"context"
	"time"

	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ClientWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, client *Client) error
	Update(ctx context.Context, client *Client) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Client, error)
	ReplaceMembers(ctx context.Context, clientID uuid.UUID, memberIDs []uuid.UUID) error
}

type ClientReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*ClientView, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]ClientView, error)
	FindPageByUserID(ctx context.Context, userID uuid.UUID, query paginate.PaginateQuery) ([]ClientView, int64, error)
}

type ClientView struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	MemberIDs []uuid.UUID
}
