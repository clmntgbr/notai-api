package subscription

import (
	"context"
	"errors"

	querysubscription "go-api/internal/application/query/subscription"
	"go-api/internal/domain/plan"
	"go-api/internal/domain/port"
	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type CreateBillingPortalCommand struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
}

type CreateBillingPortalHandler struct {
	workspaceRepo        domainworkspace.WorkspaceReadRepository
	subscriptionRepo     domainsubscription.SubscriptionReadRepository
	billingPortalGateway port.BillingPortalGateway
}

func NewCreateBillingPortalHandler(
	workspaceRepo domainworkspace.WorkspaceReadRepository,
	subscriptionRepo domainsubscription.SubscriptionReadRepository,
	billingPortalGateway port.BillingPortalGateway,
) *CreateBillingPortalHandler {
	return &CreateBillingPortalHandler{
		workspaceRepo:        workspaceRepo,
		subscriptionRepo:     subscriptionRepo,
		billingPortalGateway: billingPortalGateway,
	}
}

func (h *CreateBillingPortalHandler) Handle(ctx context.Context, cmd CreateBillingPortalCommand) (string, error) {
	workspace, err := h.workspaceRepo.FindByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return "", errors.New("failed to get workspace")
	}
	if workspace == nil || workspace.SubscriptionID == nil {
		return "", querysubscription.ErrSubscriptionNotFound
	}
	if workspace.OwnerUserID != cmd.UserID {
		return "", ErrNotWorkspaceOwner
	}

	subscriptionView, err := h.subscriptionRepo.FindByID(ctx, *workspace.SubscriptionID)
	if err != nil {
		return "", errors.New("failed to get subscription")
	}
	if subscriptionView == nil {
		return "", querysubscription.ErrSubscriptionNotFound
	}
	if subscriptionView.StripeCustomerID == "" {
		if subscriptionView.Plan != nil && subscriptionView.Plan.Slug == plan.FreePlanSlug {
			return "", ErrFreePlanBillingPortal
		}
		return "", ErrMissingStripeCustomer
	}

	url, err := h.billingPortalGateway.Create(ctx, subscriptionView.StripeCustomerID)
	if err != nil {
		return "", err
	}

	return url, nil
}
