package subscription

import (
	"context"
	"errors"
	"time"

	"go-api/internal/domain/plan"
	"go-api/internal/domain/port"
	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

var (
	ErrPlanNotFound       = errors.New("plan not found")
	ErrPlanInactive       = errors.New("plan is inactive")
	ErrFreePlanCheckout   = errors.New("free plan does not require checkout")
	ErrMissingStripePrice = errors.New("plan has no stripe price")
	ErrAlreadyOnPlan      = errors.New("workspace is already on this plan")
	ErrMissingStripeSub   = errors.New("subscription has no stripe subscription id")
)

type PreviewPlanChangeQuery struct {
	WorkspaceID uuid.UUID
	PlanID      uuid.UUID
}

type PlanChangePreview struct {
	RequiresCheckout bool
	Currency         string
	AmountDue        int64
	Subtotal         int64
	Total            int64
	ProrationDate    int64
	PeriodStart      time.Time
	PeriodEnd        time.Time
	Lines            []port.ProrationPreviewLine
	CurrentPlanID    string
	CurrentPlanSlug  string
	TargetPlanID     string
	TargetPlanSlug   string
	TargetPlanName   string
	TargetPlanPrice  float64
}

type PreviewPlanChangeHandler struct {
	workspaceRepo       domainworkspace.WorkspaceReadRepository
	planRepo            plan.PlanReadRepository
	subscriptionRepo    domainsubscription.SubscriptionReadRepository
	subscriptionGateway port.SubscriptionGateway
}

func NewPreviewPlanChangeHandler(
	workspaceRepo domainworkspace.WorkspaceReadRepository,
	planRepo plan.PlanReadRepository,
	subscriptionRepo domainsubscription.SubscriptionReadRepository,
	subscriptionGateway port.SubscriptionGateway,
) *PreviewPlanChangeHandler {
	return &PreviewPlanChangeHandler{
		workspaceRepo:       workspaceRepo,
		planRepo:            planRepo,
		subscriptionRepo:    subscriptionRepo,
		subscriptionGateway: subscriptionGateway,
	}
}

func (h *PreviewPlanChangeHandler) Handle(ctx context.Context, q PreviewPlanChangeQuery) (*PlanChangePreview, error) {
	targetPlan, err := h.planRepo.FindByID(ctx, q.PlanID)
	if err != nil {
		return nil, errors.New("failed to get plan")
	}
	if targetPlan == nil {
		return nil, ErrPlanNotFound
	}
	if !targetPlan.IsActive {
		return nil, ErrPlanInactive
	}
	if targetPlan.Slug == plan.FreePlanSlug {
		return nil, ErrFreePlanCheckout
	}
	if targetPlan.StripePriceID == "" {
		return nil, ErrMissingStripePrice
	}

	workspace, err := h.workspaceRepo.FindByID(ctx, q.WorkspaceID)
	if err != nil {
		return nil, errors.New("failed to get workspace")
	}
	if workspace == nil {
		return nil, errors.New("workspace not found")
	}

	var current *domainsubscription.SubscriptionView
	if workspace.SubscriptionID != nil {
		current, err = h.subscriptionRepo.FindByID(ctx, *workspace.SubscriptionID)
		if err != nil {
			return nil, errors.New("failed to get subscription")
		}
	}

	preview := &PlanChangePreview{
		TargetPlanID:    targetPlan.ID.String(),
		TargetPlanSlug:  targetPlan.Slug,
		TargetPlanName:  targetPlan.Name,
		TargetPlanPrice: targetPlan.Price,
		Currency:        string(targetPlan.Currency),
	}

	if current != nil {
		preview.CurrentPlanID = current.PlanID.String()
		if current.Plan != nil {
			preview.CurrentPlanSlug = current.Plan.Slug
		}
	}

	if current != nil && current.PlanID == targetPlan.ID && current.StripeSubscriptionID != "" {
		return nil, ErrAlreadyOnPlan
	}

	if !CanUpdateStripeSubscription(current) {
		preview.RequiresCheckout = true
		return preview, nil
	}

	stripeSub, err := h.subscriptionGateway.Retrieve(ctx, current.StripeSubscriptionID)
	if err != nil {
		return nil, err
	}
	if stripeSub.ItemID == "" {
		return nil, ErrMissingStripeSub
	}
	if stripeSub.PriceID == targetPlan.StripePriceID {
		return nil, ErrAlreadyOnPlan
	}

	stripePreview, err := h.subscriptionGateway.PreviewPriceChange(
		ctx,
		current.StripeSubscriptionID,
		stripeSub.ItemID,
		targetPlan.StripePriceID,
	)
	if err != nil {
		return nil, err
	}

	preview.RequiresCheckout = false
	preview.Currency = stripePreview.Currency
	preview.AmountDue = stripePreview.AmountDue
	preview.Subtotal = stripePreview.Subtotal
	preview.Total = stripePreview.Total
	preview.ProrationDate = stripePreview.ProrationDate
	preview.PeriodStart = stripePreview.PeriodStart
	preview.PeriodEnd = stripePreview.PeriodEnd
	preview.Lines = stripePreview.Lines

	return preview, nil
}

func CanUpdateStripeSubscription(subscription *domainsubscription.SubscriptionView) bool {
	if subscription == nil || subscription.StripeSubscriptionID == "" {
		return false
	}

	switch subscription.Status {
	case domainsubscription.StatusActive,
		domainsubscription.StatusPastDue,
		domainsubscription.StatusPending:
		return true
	default:
		return false
	}
}
