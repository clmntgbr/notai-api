package subscription

import (
	"context"
	"errors"
	"log"
	"time"

	"go-api/internal/domain/plan"
	"go-api/internal/domain/port"
	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type CheckoutCompletedCommand struct {
	WorkspaceID          uuid.UUID
	StripeCustomerID     string
	StripeSubscriptionID string
}

type CheckoutCompletedHandler struct {
	workspaceRepo       domainworkspace.WorkspaceWriteRepository
	planRepo            plan.PlanWriteRepository
	subscriptionRepo    domainsubscription.SubscriptionWriteRepository
	outbox              port.OutboxRepository
	subscriptionGateway port.SubscriptionGateway
	upsertInvoice       *UpsertInvoiceHandler
}

func NewCheckoutCompletedHandler(
	workspaceRepo domainworkspace.WorkspaceWriteRepository,
	planRepo plan.PlanWriteRepository,
	subscriptionRepo domainsubscription.SubscriptionWriteRepository,
	outbox port.OutboxRepository,
	subscriptionGateway port.SubscriptionGateway,
	upsertInvoice *UpsertInvoiceHandler,
) *CheckoutCompletedHandler {
	return &CheckoutCompletedHandler{
		workspaceRepo:       workspaceRepo,
		planRepo:            planRepo,
		subscriptionRepo:    subscriptionRepo,
		outbox:              outbox,
		subscriptionGateway: subscriptionGateway,
		upsertInvoice:       upsertInvoice,
	}
}

func (h *CheckoutCompletedHandler) Handle(ctx context.Context, cmd CheckoutCompletedCommand) error {
	if cmd.StripeSubscriptionID == "" {
		return errors.New("stripe subscription id is required")
	}

	workspace, err := h.workspaceRepo.GetByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return errors.New("failed to get workspace")
	}
	if workspace == nil {
		return errors.New("workspace not found")
	}

	subData, err := h.subscriptionGateway.Retrieve(ctx, cmd.StripeSubscriptionID)
	if err != nil {
		return err
	}

	targetPlan, err := h.planRepo.GetByStripePriceID(ctx, subData.PriceID)
	if err != nil {
		return errors.New("failed to get plan")
	}
	if targetPlan == nil {
		return errors.New("plan not found for stripe price id")
	}

	customerID := cmd.StripeCustomerID
	if customerID == "" {
		customerID = subData.CustomerID
	}

	status := domainsubscription.MapBillingStatus(subData.Status)

	var subscriptionEntity *domainsubscription.Subscription
	if workspace.SubscriptionID != nil {
		subscriptionEntity, err = h.subscriptionRepo.GetByID(ctx, *workspace.SubscriptionID)
		if err != nil {
			return errors.New("failed to get subscription")
		}
	}

	wasFree := subscriptionEntity == nil
	if subscriptionEntity != nil {
		currentPlan, err := h.planRepo.GetByID(ctx, subscriptionEntity.PlanID)
		if err != nil {
			return errors.New("failed to get current plan")
		}
		wasFree = currentPlan == nil || currentPlan.Slug == plan.FreePlanSlug || subscriptionEntity.StripeSubscriptionID == ""
	}

	startDate := time.Now().UTC()
	endDate := startDate
	if !subData.CurrentPeriodStart.IsZero() {
		startDate = subData.CurrentPeriodStart
	}
	if !subData.CurrentPeriodEnd.IsZero() {
		endDate = subData.CurrentPeriodEnd
	}

	quotaPeriodStart := time.Time{}
	if wasFree {
		if !subData.CurrentPeriodStart.IsZero() {
			quotaPeriodStart = subData.CurrentPeriodStart
		} else {
			quotaPeriodStart = time.Now().UTC()
		}
	}

	if err := h.workspaceRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if subscriptionEntity == nil {
			subscriptionEntity = domainsubscription.NewSubscription(targetPlan.ID, status, startDate, endDate)
			subscriptionEntity.ApplyUpdate(
				targetPlan.ID,
				status,
				customerID,
				subData.ID,
				startDate,
				endDate,
				subData.CancelAtPeriodEnd,
				quotaPeriodStart,
			)
			if err := h.subscriptionRepo.Save(txCtx, subscriptionEntity); err != nil {
				return errors.New("failed to create subscription")
			}
		} else {
			if quotaPeriodStart.IsZero() {
				quotaPeriodStart = subscriptionEntity.QuotaPeriodStart
			}
			subscriptionEntity.ApplyUpdate(
				targetPlan.ID,
				status,
				customerID,
				subData.ID,
				startDate,
				endDate,
				subData.CancelAtPeriodEnd,
				quotaPeriodStart,
			)
			if err := h.subscriptionRepo.Update(txCtx, subscriptionEntity); err != nil {
				return errors.New("failed to update subscription")
			}
		}

		events := subscriptionEntity.PullEvents()

		if workspace.SubscriptionID == nil || *workspace.SubscriptionID != subscriptionEntity.ID {
			workspace.AssignSubscription(subscriptionEntity.ID)
			if err := h.workspaceRepo.Update(txCtx, workspace); err != nil {
				return errors.New("failed to link subscription to workspace")
			}
			events = append(events, workspace.PullEvents()...)
		}

		return h.outbox.StoreEvents(txCtx, events)
	}); err != nil {
		return err
	}

	// invoice.payment_succeeded often arrives before checkout.session.completed.
	// Sync the latest Stripe invoice now so local billing is complete even when
	// the earlier webhook returned "subscription not linked yet" (no reliable retry on stripe listen).
	return h.syncLatestInvoice(ctx, cmd.StripeSubscriptionID)
}

func (h *CheckoutCompletedHandler) syncLatestInvoice(ctx context.Context, stripeSubscriptionID string) error {
	if h.upsertInvoice == nil || h.subscriptionGateway == nil {
		return nil
	}

	invoiceData, err := h.subscriptionGateway.RetrieveLatestInvoice(ctx, stripeSubscriptionID)
	if err != nil {
		return err
	}
	if invoiceData == nil || invoiceData.ID == "" {
		log.Printf(
			"checkout completed: no invoice found yet for stripeSubscriptionID=%s",
			stripeSubscriptionID,
		)
		return nil
	}

	cmd := UpsertInvoiceCommand{
		StripeInvoiceID:      invoiceData.ID,
		StripeCustomerID:     invoiceData.CustomerID,
		StripeSubscriptionID: invoiceData.SubscriptionID,
		Number:               invoiceData.Number,
		Status:               invoiceData.Status,
		Currency:             invoiceData.Currency,
		AmountDue:            invoiceData.AmountDue,
		AmountPaid:           invoiceData.AmountPaid,
		Total:                invoiceData.Total,
		HostedInvoiceURL:     invoiceData.HostedInvoiceURL,
		InvoicePDF:           invoiceData.InvoicePDF,
		BillingReason:        invoiceData.BillingReason,
		Description:          invoiceData.Description,
		AttemptCount:         invoiceData.AttemptCount,
		PeriodStart:          invoiceData.PeriodStart,
		PeriodEnd:            invoiceData.PeriodEnd,
		PaidAt:               invoiceData.PaidAt,
		StripeCreatedAt:      invoiceData.CreatedAt,
		StripeEventID:        "checkout.session.completed:" + stripeSubscriptionID,
	}
	if invoiceData.SubscriptionID == "" {
		cmd.StripeSubscriptionID = stripeSubscriptionID
	}
	if invoiceData.Status == "paid" {
		cmd.PaymentOutcome = "succeeded"
	}

	log.Printf(
		"checkout completed: syncing latest invoice stripeInvoiceID=%s stripeSubscriptionID=%s status=%s",
		cmd.StripeInvoiceID,
		cmd.StripeSubscriptionID,
		cmd.Status,
	)
	return h.upsertInvoice.Handle(ctx, cmd)
}
