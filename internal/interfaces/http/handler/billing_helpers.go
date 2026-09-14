package handler

import (
	"errors"

	queryclient "go-api/internal/application/query/client"
	httpctx "go-api/internal/interfaces/http/context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

var errActiveClientRequired = errors.New("active client is required")

func resolveCurrentClientWorkspace(
	c fiber.Ctx,
	getClient clientGetByIDHandler,
) (uuid.UUID, uuid.UUID, error) {
	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, errActiveClientRequired
	}

	client, err := getClient.Handle(c.Context(), queryclient.GetClientByIDQuery{ID: clientID})
	if err != nil {
		if err.Error() == "client not found" {
			return uuid.Nil, uuid.Nil, err
		}
		return uuid.Nil, uuid.Nil, errors.New("failed to get client")
	}
	if client == nil {
		return uuid.Nil, uuid.Nil, errors.New("client not found")
	}
	return client.ID, client.WorkspaceID, nil
}

func respondResolveClientWorkspaceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, errActiveClientRequired):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Active client is required",
		})
	case err != nil && err.Error() == "client not found":
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Client not found",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get client",
		})
	}
}
