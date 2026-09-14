package subscription

import (
	"context"
	"errors"
	"log"
	"time"

	"go-api/internal/domain/event"
	domaininvoice "go-api/internal/domain/invoice"
	"go-api/internal/domain/port"
	domainsubscription "go-api/internal/domain/subscription"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type UpsertInvoiceCommand struct {
	StripeInvoiceID      string
	StripeCustomerID     string
	StripeSubscriptionID string
	Number               string
	Status               string
	Currency             string
	AmountDue            int64
	AmountPaid           int64
	Total                int64
	HostedInvoiceURL     string
	InvoicePDF           string
	BillingReason        string
	Description          string
	AttemptCount         int64
	PeriodStart          time.Time
	PeriodEnd            time.Time
	PaidAt               *time.Time
	StripeCreatedAt      time.Time
	StripeEventID        string
	PaymentOutcome       string
}

type UpsertInvoiceHandler struct {
	invoiceRepo      domaininvoice.InvoiceWriteRepository
	subscriptionRepo domainsubscription.SubscriptionWriteRepository
	workspaceRepo    domainworkspace.WorkspaceWriteRepository
	outbox           port.OutboxRepository
}

func NewUpsertInvoiceHandler(
	invoiceRepo domaininvoice.InvoiceWriteRepository,
	subscriptionRepo domainsubscription.SubscriptionWriteRepository,
	workspaceRepo domainworkspace.WorkspaceWriteRepository,
	outbox port.OutboxRepository,
) *UpsertInvoiceHandler {
	return &UpsertInvoiceHandler{
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		workspaceRepo:    workspaceRepo,
		outbox:           outbox,
	}
}

func (h *UpsertInvoiceHandler) Handle(ctx context.Context, cmd UpsertInvoiceCommand) error {
	log.Printf(
		"upsert invoice: start stripeInvoiceID=%s stripeSubscriptionID=%s stripeCustomerID=%s status=%s total=%d",
		cmd.StripeInvoiceID,
		cmd.StripeSubscriptionID,
		cmd.StripeCustomerID,
		cmd.Status,
		cmd.Total,
	)

	if cmd.StripeInvoiceID == "" {
		log.Printf("upsert invoice: skip, missing stripeInvoiceID")
		return nil
	}

	workspaceID, subscriptionID, err := h.resolveOwner(ctx, cmd.StripeSubscriptionID, cmd.StripeCustomerID)
	if err != nil {
		log.Printf("upsert invoice: resolve owner failed: %v", err)
		return err
	}
	if workspaceID == uuid.Nil {
		log.Printf(
			"upsert invoice: subscription not linked yet stripeSubscriptionID=%s stripeCustomerID=%s",
			cmd.StripeSubscriptionID,
			cmd.StripeCustomerID,
		)
		return ErrStripeSubscriptionNotLinked
	}

	subscriptionIDValue := ""
	if subscriptionID != nil {
		subscriptionIDValue = subscriptionID.String()
	}
	log.Printf("upsert invoice: resolved workspaceID=%s subscriptionID=%s", workspaceID, subscriptionIDValue)

	invoiceEntity, err := h.invoiceRepo.GetByStripeInvoiceID(ctx, cmd.StripeInvoiceID)
	if err != nil {
		return errors.New("failed to get invoice")
	}

	isNew := invoiceEntity == nil
	if isNew {
		invoiceEntity = domaininvoice.NewInvoice(workspaceID, subscriptionID, cmd.StripeInvoiceID)
	} else {
		invoiceEntity.WorkspaceID = workspaceID
		invoiceEntity.SubscriptionID = subscriptionID
	}

	invoiceEntity.ApplyStripeSnapshot(
		cmd.StripeCustomerID,
		cmd.StripeSubscriptionID,
		cmd.Number,
		cmd.Status,
		cmd.Currency,
		cmd.AmountDue,
		cmd.AmountPaid,
		cmd.Total,
		cmd.HostedInvoiceURL,
		cmd.InvoicePDF,
		cmd.BillingReason,
		cmd.Description,
		cmd.AttemptCount,
		cmd.PeriodStart,
		cmd.PeriodEnd,
		cmd.PaidAt,
		cmd.StripeCreatedAt,
	)
	if isNew {
		invoiceEntity.RaiseCreated()
	} else {
		invoiceEntity.RaiseUpdated()
	}
	switch cmd.PaymentOutcome {
	case "succeeded":
		invoiceEntity.RaisePaymentSucceeded(event.DeterministicID("stripe", cmd.StripeEventID, "invoice.paymentSucceeded"))
	case "failed":
		invoiceEntity.RaisePaymentFailed(event.DeterministicID("stripe", cmd.StripeEventID, "invoice.paymentFailed"))
	}

	err = h.invoiceRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.invoiceRepo.UpsertByStripeInvoiceID(txCtx, invoiceEntity); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, invoiceEntity.PullEvents())
	})
	if err != nil {
		log.Printf("upsert invoice: db upsert failed stripeInvoiceID=%s: %v", cmd.StripeInvoiceID, err)
		return errors.New("failed to upsert invoice")
	}

	log.Printf("upsert invoice: ok id=%s stripeInvoiceID=%s", invoiceEntity.ID, invoiceEntity.StripeInvoiceID)
	return nil
}

func (h *UpsertInvoiceHandler) resolveOwner(
	ctx context.Context,
	stripeSubscriptionID string,
	stripeCustomerID string,
) (uuid.UUID, *uuid.UUID, error) {
	subscriptionEntity, err := findSubscriptionByStripeIDs(
		ctx,
		h.subscriptionRepo,
		stripeSubscriptionID,
		stripeCustomerID,
	)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if subscriptionEntity == nil {
		return uuid.Nil, nil, nil
	}

	workspace, err := h.workspaceRepo.GetBySubscriptionID(ctx, subscriptionEntity.ID)
	if err != nil {
		return uuid.Nil, nil, errors.New("failed to get workspace by subscription")
	}
	if workspace == nil {
		log.Printf("upsert invoice: no workspace linked to subscriptionID=%s", subscriptionEntity.ID)
		return uuid.Nil, nil, nil
	}

	subscriptionID := subscriptionEntity.ID
	return workspace.ID, &subscriptionID, nil
}
