package handler

import (
	usercmd "go-api/internal/application/command/user"
	queryuser "go-api/internal/application/query/user"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type UserHandler struct {
	getUserByIDHandler      userGetByIDHandler
	setCurrentClientHandler userSetCurrentClientHandler
}

func NewUserHandler(
	getUserByIDHandler userGetByIDHandler,
	setCurrentClientHandler userSetCurrentClientHandler,
) *UserHandler {
	return &UserHandler{
		getUserByIDHandler:      getUserByIDHandler,
		setCurrentClientHandler: setCurrentClientHandler,
	}
}

func (h *UserHandler) GetUser(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	view, err := h.getUserByIDHandler.Handle(c.Context(), queryuser.GetUserByIDQuery{ID: user.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewUserDetailResponseFromView(*view))
}

func (h *UserHandler) SetCurrentClient(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var req dto.SetCurrentClientRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}

	err = h.setCurrentClientHandler.Handle(c.Context(), usercmd.SetCurrentClientCommand{
		UserID:   user.ID,
		ClientID: clientID,
	})
	if err != nil {
		if err.Error() == "client not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Client not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to set current client"})
	}

	view, err := h.getUserByIDHandler.Handle(c.Context(), queryuser.GetUserByIDQuery{ID: user.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get user"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewUserDetailResponseFromView(*view))
}
