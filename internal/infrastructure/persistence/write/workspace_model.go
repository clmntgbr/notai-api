package write

import (
	"time"

	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type WorkspaceModel struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"`
	OwnerUserID    uuid.UUID  `gorm:"column:owner_user_id"`
	SubscriptionID *uuid.UUID `gorm:"column:subscription_id"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (WorkspaceModel) TableName() string {
	return "workspaces"
}

func workspaceModelFromDomain(w *domainworkspace.Workspace) *WorkspaceModel {
	return &WorkspaceModel{
		ID:             w.ID,
		OwnerUserID:    w.OwnerUserID,
		SubscriptionID: w.SubscriptionID,
		CreatedAt:      w.CreatedAt,
		UpdatedAt:      w.UpdatedAt,
	}
}

func workspaceDomainFromModel(m *WorkspaceModel) *domainworkspace.Workspace {
	return &domainworkspace.Workspace{
		ID:             m.ID,
		OwnerUserID:    m.OwnerUserID,
		SubscriptionID: m.SubscriptionID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
