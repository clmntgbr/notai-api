package subscription

import "errors"

var (
	ErrStripeSubscriptionNotLinked = errors.New("stripe subscription not linked yet")
	ErrMissingStripeCustomer       = errors.New("workspace has no stripe customer id")
	ErrFreePlanBillingPortal       = errors.New("billing portal is not available on free plan")
	ErrNotWorkspaceOwner           = errors.New("only the workspace owner can manage billing")
)
