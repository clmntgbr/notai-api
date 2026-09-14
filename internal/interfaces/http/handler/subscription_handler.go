package handler

import (
	"errors"
	"log"

	cmdsubscription "go-api/internal/application/command/subscription"
	querysubscription "go-api/internal/application/query/subscription"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	getClientHandler              clientGetByIDHandler
	getCurrentSubscriptionHandler subscriptionGetCurrentHandler
	getQuotaUsageHandler          subscriptionGetQuotaHandler
	previewPlanChangeHandler      subscriptionPreviewPlanChangeHandler
	createSubscriptionHandler     subscriptionCreateHandler
	createBillingPortalHandler    subscriptionCreateBillingPortalHandler
}

func NewSubscriptionHandler(
	getClientHandler clientGetByIDHandler,
	getCurrentSubscriptionHandler subscriptionGetCurrentHandler,
	getQuotaUsageHandler subscriptionGetQuotaHandler,
	previewPlanChangeHandler subscriptionPreviewPlanChangeHandler,
	createSubscriptionHandler subscriptionCreateHandler,
	createBillingPortalHandler subscriptionCreateBillingPortalHandler,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		getClientHandler:              getClientHandler,
		getCurrentSubscriptionHandler: getCurrentSubscriptionHandler,
		getQuotaUsageHandler:          getQuotaUsageHandler,
		previewPlanChangeHandler:      previewPlanChangeHandler,
		createSubscriptionHandler:     createSubscriptionHandler,
		createBillingPortalHandler:    createBillingPortalHandler,
	}
}

func (h *SubscriptionHandler) GetSubscription(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	_, workspaceID, err := resolveCurrentClientWorkspace(c, h.getClientHandler)
	if err != nil {
		return respondResolveClientWorkspaceError(c, err)
	}

	view, err := h.getCurrentSubscriptionHandler.Handle(c.Context(), querysubscription.GetCurrentSubscriptionQuery{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if errors.Is(err, querysubscription.ErrSubscriptionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Subscription not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get subscription",
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewSubscriptionResponse(view))
}

func (h *SubscriptionHandler) GetQuota(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	clientID, _, err := resolveCurrentClientWorkspace(c, h.getClientHandler)
	if err != nil {
		return respondResolveClientWorkspaceError(c, err)
	}

	usage, err := h.getQuotaUsageHandler.Handle(c.Context(), querysubscription.GetQuotaUsageQuery{
		ClientID: clientID,
	})
	if err != nil {
		if errors.Is(err, querysubscription.ErrSubscriptionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Subscription not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get quota usage",
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewQuotaUsageResponse(usage))
}

func (h *SubscriptionHandler) PreviewSubscription(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	_, workspaceID, err := resolveCurrentClientWorkspace(c, h.getClientHandler)
	if err != nil {
		return respondResolveClientWorkspaceError(c, err)
	}

	var request dto.PreviewSubscriptionRequest
	if err := validation.BindBody(c, &request); err != nil {
		return err
	}

	planID, err := uuid.Parse(request.PlanID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid planId",
		})
	}

	preview, err := h.previewPlanChangeHandler.Handle(c.Context(), querysubscription.PreviewPlanChangeQuery{
		WorkspaceID: workspaceID,
		PlanID:      planID,
	})
	if err != nil {
		switch {
		case errors.Is(err, querysubscription.ErrPlanNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Plan not found",
			})
		case errors.Is(err, querysubscription.ErrPlanInactive),
			errors.Is(err, querysubscription.ErrFreePlanCheckout),
			errors.Is(err, querysubscription.ErrMissingStripePrice),
			errors.Is(err, querysubscription.ErrAlreadyOnPlan):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			log.Printf("subscription: preview failed for user %s workspace %s: %v", user.ID, workspaceID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to preview plan change",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewPlanChangePreviewResponse(preview))
}

func (h *SubscriptionHandler) CreateSubscription(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	_, workspaceID, err := resolveCurrentClientWorkspace(c, h.getClientHandler)
	if err != nil {
		return respondResolveClientWorkspaceError(c, err)
	}

	var request dto.CreateSubscriptionRequest
	if err := validation.BindBody(c, &request); err != nil {
		return err
	}

	planID, err := uuid.Parse(request.PlanID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid planId",
		})
	}

	result, err := h.createSubscriptionHandler.Handle(c.Context(), cmdsubscription.CreateSubscriptionCommand{
		WorkspaceID:   workspaceID,
		UserID:        user.ID,
		PlanID:        planID,
		ProrationDate: request.ProrationDate,
	})
	if err != nil {
		switch {
		case errors.Is(err, querysubscription.ErrPlanNotFound),
			errors.Is(err, querysubscription.ErrSubscriptionNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": err.Error(),
			})
		case errors.Is(err, cmdsubscription.ErrNotWorkspaceOwner):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Only the workspace owner can manage billing",
			})
		case errors.Is(err, querysubscription.ErrPlanInactive),
			errors.Is(err, querysubscription.ErrFreePlanCheckout),
			errors.Is(err, querysubscription.ErrMissingStripePrice),
			errors.Is(err, querysubscription.ErrAlreadyOnPlan):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			log.Printf("subscription: change plan failed for user %s workspace %s: %v", user.ID, workspaceID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Internal server error",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewChangeSubscriptionResponse(result))
}

func (h *SubscriptionHandler) CreateBillingPortal(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	_, workspaceID, err := resolveCurrentClientWorkspace(c, h.getClientHandler)
	if err != nil {
		return respondResolveClientWorkspaceError(c, err)
	}

	url, err := h.createBillingPortalHandler.Handle(c.Context(), cmdsubscription.CreateBillingPortalCommand{
		WorkspaceID: workspaceID,
		UserID:      user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, querysubscription.ErrSubscriptionNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Subscription not found",
			})
		case errors.Is(err, cmdsubscription.ErrNotWorkspaceOwner):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Only the workspace owner can manage billing",
			})
		case errors.Is(err, cmdsubscription.ErrMissingStripeCustomer):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Workspace has no Stripe customer",
			})
		case errors.Is(err, cmdsubscription.ErrFreePlanBillingPortal):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": err.Error(),
			})
		default:
			log.Printf("subscription: billing portal failed for user %s workspace %s: %v", user.ID, workspaceID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Internal server error",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewCheckoutSessionResponse(url))
}
