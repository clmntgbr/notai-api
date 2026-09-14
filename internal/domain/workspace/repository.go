package workspace

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type WorkspaceWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, workspace *Workspace) error
	Update(ctx context.Context, workspace *Workspace) error
	GetByID(ctx context.Context, id uuid.UUID) (*Workspace, error)
	GetByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (*Workspace, error)
	GetBySubscriptionID(ctx context.Context, subscriptionID uuid.UUID) (*Workspace, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*Workspace, error)
}

type WorkspaceReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*WorkspaceView, error)
}

type WorkspaceView struct {
	ID             uuid.UUID
	OwnerUserID    uuid.UUID
	SubscriptionID *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
