package handler

import (
	"strings"

	clientcmd "go-api/internal/application/command/client"
	queryclient "go-api/internal/application/query/client"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ClientHandler struct {
	createHandler       clientCreateHandler
	updateHandler       clientUpdateHandler
	deleteHandler       clientDeleteHandler
	addMemberHandler    clientAddMemberHandler
	removeMemberHandler clientRemoveMemberHandler
	getByIDHandler      clientGetByIDHandler
	listByUserHandler   clientListByUserHandler
}

func NewClientHandler(
	createHandler clientCreateHandler,
	updateHandler clientUpdateHandler,
	deleteHandler clientDeleteHandler,
	addMemberHandler clientAddMemberHandler,
	removeMemberHandler clientRemoveMemberHandler,
	getByIDHandler clientGetByIDHandler,
	listByUserHandler clientListByUserHandler,
) *ClientHandler {
	return &ClientHandler{
		createHandler:       createHandler,
		updateHandler:       updateHandler,
		deleteHandler:       deleteHandler,
		addMemberHandler:    addMemberHandler,
		removeMemberHandler: removeMemberHandler,
		getByIDHandler:      getByIDHandler,
		listByUserHandler:   listByUserHandler,
	}
}

func (h *ClientHandler) List(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var query paginate.PaginateQuery
	if err := c.Bind().Query(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	sortBy, orderBy := query.SortBy, query.OrderBy
	query.Normalize()
	if sortBy == "" {
		query.SortBy = "created_at"
	}
	if orderBy == "" {
		query.OrderBy = paginate.OrderByAsc
	}

	views, total, err := h.listByUserHandler.Handle(c.Context(), queryclient.ListClientsByUserQuery{
		UserID: user.ID,
		Query:  query,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list clients"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewClientListResponseFromViews(views, user.CurrentClientID),
		int(total),
		query,
	))
}

func (h *ClientHandler) Create(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	var req dto.CreateClientRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	client, err := h.createHandler.Handle(c.Context(), clientcmd.CreateClientCommand{
		Name:          req.Name,
		CreatorUserID: user.ID,
	})
	if err != nil {
		if err.Error() == "workspace not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Workspace not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create client"})
	}

	activeID := client.ID
	return c.Status(fiber.StatusCreated).JSON(
		presenter.NewClientDetailResponseFromEntity(*client, &activeID),
	)
}

func (h *ClientHandler) GetByID(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}

	view, err := h.getByIDHandler.Handle(c.Context(), queryclient.GetClientByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "client not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Client not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get client"})
	}

	return c.Status(fiber.StatusOK).JSON(
		presenter.NewClientDetailResponseFromView(*view, user.CurrentClientID),
	)
}

func (h *ClientHandler) Update(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}

	var req dto.UpdateClientRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	err = h.updateHandler.Handle(c.Context(), clientcmd.UpdateClientCommand{
		ID:   id,
		Name: req.Name,
	})
	if err != nil {
		if err.Error() == "client not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Client not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update client"})
	}

	view, err := h.getByIDHandler.Handle(c.Context(), queryclient.GetClientByIDQuery{ID: id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get client"})
	}

	return c.Status(fiber.StatusOK).JSON(
		presenter.NewClientDetailResponseFromView(*view, user.CurrentClientID),
	)
}

func (h *ClientHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}

	if err := h.deleteHandler.Handle(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete client"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ClientHandler) AddMember(c fiber.Ctx) error {
	actor, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}

	var req dto.AddClientMemberRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid user id"})
	}

	err = h.addMemberHandler.Handle(c.Context(), clientcmd.AddClientMemberCommand{
		ClientID:    clientID,
		UserID:      userID,
		ActorUserID: actor.ID,
	})
	if err != nil {
		if handled, quotaErr := respondQuotaError(c, err); handled {
			return quotaErr
		}
		if err.Error() == "client not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Client not found"})
		}
		if err.Error() == "user not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to add member"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ClientHandler) RemoveMember(c fiber.Ctx) error {
	clientID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid client id"})
	}
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid user id"})
	}

	err = h.removeMemberHandler.Handle(c.Context(), clientcmd.RemoveClientMemberCommand{
		ClientID: clientID,
		UserID:   userID,
	})
	if err != nil {
		if err.Error() == "client not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Client not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to remove member"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
