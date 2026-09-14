package presenter

import (
	"time"

	domainworkspace "go-api/internal/domain/workspace"
)

type WorkspaceResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	OwnerUserID    string    `json:"ownerUserId"`
	SubscriptionID *string   `json:"subscriptionId"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func NewWorkspaceResponse(view *domainworkspace.WorkspaceView) WorkspaceResponse {
	var subscriptionID *string
	if view.SubscriptionID != nil {
		value := view.SubscriptionID.String()
		subscriptionID = &value
	}

	return WorkspaceResponse{
		ID:             view.ID.String(),
		Name:           view.Name,
		OwnerUserID:    view.OwnerUserID.String(),
		SubscriptionID: subscriptionID,
		CreatedAt:      view.CreatedAt,
		UpdatedAt:      view.UpdatedAt,
	}
}
