package write

import (
	"context"
	"errors"

	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workspaceWriteRepository struct {
	db *gorm.DB
}

func NewWorkspaceWriteRepository(db *gorm.DB) domainworkspace.WorkspaceWriteRepository {
	return &workspaceWriteRepository{db: db}
}

func (r *workspaceWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *workspaceWriteRepository) Save(ctx context.Context, workspace *domainworkspace.Workspace) error {
	return DBWithContext(ctx, r.db).Create(workspaceModelFromDomain(workspace)).Error
}

func (r *workspaceWriteRepository) Update(ctx context.Context, workspace *domainworkspace.Workspace) error {
	return DBWithContext(ctx, r.db).Save(workspaceModelFromDomain(workspace)).Error
}

func (r *workspaceWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainworkspace.Workspace, error) {
	var model WorkspaceModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return workspaceDomainFromModel(&model), nil
}

func (r *workspaceWriteRepository) GetByOwnerUserID(
	ctx context.Context,
	ownerUserID uuid.UUID,
) (*domainworkspace.Workspace, error) {
	var model WorkspaceModel
	err := DBWithContext(ctx, r.db).Where("owner_user_id = ?", ownerUserID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return workspaceDomainFromModel(&model), nil
}

func (r *workspaceWriteRepository) GetBySubscriptionID(
	ctx context.Context,
	subscriptionID uuid.UUID,
) (*domainworkspace.Workspace, error) {
	var model WorkspaceModel
	err := DBWithContext(ctx, r.db).Where("subscription_id = ?", subscriptionID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return workspaceDomainFromModel(&model), nil
}

func (r *workspaceWriteRepository) GetByStripeCustomerID(
	ctx context.Context,
	stripeCustomerID string,
) (*domainworkspace.Workspace, error) {
	db := DBWithContext(ctx, r.db)
	var ids []uuid.UUID
	err := db.
		Table("subscriptions").
		Where("stripe_customer_id = ?", stripeCustomerID).
		Limit(1).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return r.GetBySubscriptionID(ctx, ids[0])
}
