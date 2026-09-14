package subscription

import (
	"context"
	"errors"

	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")

type GetCurrentSubscriptionQuery struct {
	WorkspaceID uuid.UUID
}

type GetCurrentSubscriptionHandler struct {
	workspaceRepo    domainworkspace.WorkspaceReadRepository
	subscriptionRepo domainsubscription.SubscriptionReadRepository
}

func NewGetCurrentSubscriptionHandler(
	workspaceRepo domainworkspace.WorkspaceReadRepository,
	subscriptionRepo domainsubscription.SubscriptionReadRepository,
) *GetCurrentSubscriptionHandler {
	return &GetCurrentSubscriptionHandler{
		workspaceRepo:    workspaceRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

func (h *GetCurrentSubscriptionHandler) Handle(
	ctx context.Context,
	q GetCurrentSubscriptionQuery,
) (*domainsubscription.SubscriptionView, error) {
	workspace, err := h.workspaceRepo.FindByID(ctx, q.WorkspaceID)
	if err != nil {
		return nil, errors.New("failed to get workspace")
	}
	if workspace == nil || workspace.SubscriptionID == nil {
		return nil, ErrSubscriptionNotFound
	}

	view, err := h.subscriptionRepo.FindByID(ctx, *workspace.SubscriptionID)
	if err != nil {
		return nil, errors.New("failed to get subscription")
	}
	if view == nil {
		return nil, ErrSubscriptionNotFound
	}

	return view, nil
}
