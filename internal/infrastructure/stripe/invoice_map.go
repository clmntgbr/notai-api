package stripe

import (
	"go-api/internal/domain/port"

	"github.com/stripe/stripe-go/v82"
)

func MapInvoiceData(invoice *stripe.Invoice) *port.InvoiceData {
	if invoice == nil {
		return nil
	}

	data := &port.InvoiceData{
		ID:               invoice.ID,
		CustomerID:       customerIDFromInvoice(invoice),
		SubscriptionID:   subscriptionIDFromInvoice(invoice),
		Number:           invoice.Number,
		Status:           string(invoice.Status),
		Currency:         string(invoice.Currency),
		AmountDue:        invoice.AmountDue,
		AmountPaid:       invoice.AmountPaid,
		Total:            invoice.Total,
		HostedInvoiceURL: invoice.HostedInvoiceURL,
		InvoicePDF:       invoice.InvoicePDF,
		BillingReason:    string(invoice.BillingReason),
		Description:      invoiceDescription(invoice),
		AttemptCount:     invoice.AttemptCount,
		PeriodStart:      unixToTime(invoice.PeriodStart),
		PeriodEnd:        unixToTime(invoice.PeriodEnd),
		CreatedAt:        unixToTime(invoice.Created),
	}
	if invoice.StatusTransitions != nil {
		if paidAt := unixToTime(invoice.StatusTransitions.PaidAt); !paidAt.IsZero() {
			data.PaidAt = &paidAt
		}
	}
	return data
}

func invoiceDescription(invoice *stripe.Invoice) string {
	if invoice.Lines == nil || len(invoice.Lines.Data) == 0 {
		return ""
	}
	return invoice.Lines.Data[0].Description
}

func subscriptionIDFromInvoice(invoice *stripe.Invoice) string {
	if invoice.Parent != nil &&
		invoice.Parent.SubscriptionDetails != nil &&
		invoice.Parent.SubscriptionDetails.Subscription != nil &&
		invoice.Parent.SubscriptionDetails.Subscription.ID != "" {
		return invoice.Parent.SubscriptionDetails.Subscription.ID
	}

	if invoice.Lines != nil {
		for _, line := range invoice.Lines.Data {
			if line == nil || line.Parent == nil || line.Parent.SubscriptionItemDetails == nil {
				continue
			}
			if line.Parent.SubscriptionItemDetails.Subscription != "" {
				return line.Parent.SubscriptionItemDetails.Subscription
			}
		}
	}

	return ""
}

func customerIDFromInvoice(invoice *stripe.Invoice) string {
	if invoice.Customer != nil && invoice.Customer.ID != "" {
		return invoice.Customer.ID
	}
	return ""
}
