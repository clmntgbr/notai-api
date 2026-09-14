package handler

import (
	"errors"

	queryworkspace "go-api/internal/application/query/workspace"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type WorkspaceHandler struct {
	getOwnedWorkspaceHandler workspaceGetOwnedHandler
}

func NewWorkspaceHandler(getOwnedWorkspaceHandler workspaceGetOwnedHandler) *WorkspaceHandler {
	return &WorkspaceHandler{getOwnedWorkspaceHandler: getOwnedWorkspaceHandler}
}

func (h *WorkspaceHandler) GetMe(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	view, err := h.getOwnedWorkspaceHandler.Handle(c.Context(), queryworkspace.GetOwnedWorkspaceQuery{
		OwnerUserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, queryworkspace.ErrWorkspaceNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Workspace not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get workspace",
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewWorkspaceResponse(view))
}
