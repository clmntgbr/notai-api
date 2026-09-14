package subscription

import (
	"context"
	"errors"

	domainsubscription "go-api/internal/domain/subscription"
)

func findSubscriptionByStripeIDs(
	ctx context.Context,
	repo domainsubscription.SubscriptionWriteRepository,
	stripeSubscriptionID string,
	stripeCustomerID string,
) (*domainsubscription.Subscription, error) {
	if stripeSubscriptionID != "" {
		subscriptionEntity, err := repo.GetByStripeSubscriptionID(ctx, stripeSubscriptionID)
		if err != nil {
			return nil, errors.New("failed to get subscription")
		}
		if subscriptionEntity != nil {
			return subscriptionEntity, nil
		}
	}

	// Customer fallback only when the row has no Stripe subscription yet (Free → paid race).
	if stripeCustomerID == "" {
		return nil, nil
	}
	subscriptionEntity, err := repo.GetByStripeCustomerID(ctx, stripeCustomerID)
	if err != nil {
		return nil, errors.New("failed to get subscription by customer")
	}
	if subscriptionEntity == nil {
		return nil, nil
	}
	if subscriptionEntity.StripeSubscriptionID != "" &&
		stripeSubscriptionID != "" &&
		subscriptionEntity.StripeSubscriptionID != stripeSubscriptionID {
		return nil, nil
	}
	return subscriptionEntity, nil
}
