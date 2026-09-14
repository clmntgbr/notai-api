package read

import (
	"context"
	"errors"
	"time"

	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type workspaceRow struct {
	ID             uuid.UUID
	Name           string
	OwnerUserID    uuid.UUID
	SubscriptionID *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (workspaceRow) TableName() string { return "workspaces" }

type workspaceReadRepository struct {
	db *gorm.DB
}

func NewWorkspaceReadRepository(db *gorm.DB) domainworkspace.WorkspaceReadRepository {
	return &workspaceReadRepository{db: db}
}

func (r *workspaceReadRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*domainworkspace.WorkspaceView, error) {
	var row workspaceRow
	err := r.db.WithContext(ctx).
		Select("id", "name", "owner_user_id", "subscription_id", "created_at", "updated_at").
		First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toWorkspaceView(row), nil
}

func (r *workspaceReadRepository) FindByOwnerUserID(
	ctx context.Context,
	ownerUserID uuid.UUID,
) (*domainworkspace.WorkspaceView, error) {
	var row workspaceRow
	err := r.db.WithContext(ctx).
		Select("id", "name", "owner_user_id", "subscription_id", "created_at", "updated_at").
		Where("owner_user_id = ?", ownerUserID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toWorkspaceView(row), nil
}

func toWorkspaceView(row workspaceRow) *domainworkspace.WorkspaceView {
	return &domainworkspace.WorkspaceView{
		ID:             row.ID,
		Name:           row.Name,
		OwnerUserID:    row.OwnerUserID,
		SubscriptionID: row.SubscriptionID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
