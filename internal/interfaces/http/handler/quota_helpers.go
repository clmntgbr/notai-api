package handler

import (
	"errors"

	cmdquota "go-api/internal/application/command/quota"
	querysubscription "go-api/internal/application/query/subscription"

	"github.com/gofiber/fiber/v3"
)

func respondQuotaError(c fiber.Ctx, err error) (bool, error) {
	if err == nil {
		return false, nil
	}

	switch {
	case errors.Is(err, querysubscription.ErrSubscriptionNotFound):
		return true, c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "No active subscription found",
		})
	case errors.Is(err, cmdquota.ErrMemberQuotaExceeded),
		errors.Is(err, cmdquota.ErrCampaignQuotaExceeded),
		errors.Is(err, cmdquota.ErrVerificationQuotaExceeded),
		errors.Is(err, cmdquota.ErrConcurrentQuotaExceeded),
		errors.Is(err, cmdquota.ErrFileSizeQuotaExceeded),
		errors.Is(err, cmdquota.ErrVideoAnalysisNotAllowed),
		errors.Is(err, cmdquota.ErrBatchUploadQuotaExceeded),
		errors.Is(err, cmdquota.ErrStorageQuotaExceeded):
		return true, c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": err.Error(),
		})
	default:
		return false, nil
	}
}
