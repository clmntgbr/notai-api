package handler

import (
	queryactivity "go-api/internal/application/query/activity"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type ActivityHandler struct {
	listHandler activityListByClientHandler
}

func NewActivityHandler(listHandler activityListByClientHandler) *ActivityHandler {
	return &ActivityHandler{listHandler: listHandler}
}

func (h *ActivityHandler) List(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	var listQuery paginate.PaginateQuery
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	sortBy, orderBy := listQuery.SortBy, listQuery.OrderBy
	listQuery.Normalize()
	if sortBy == "" {
		listQuery.SortBy = "occurred_at"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByDesc
	}

	views, total, err := h.listHandler.Handle(c.Context(), queryactivity.ListByClientQuery{
		ClientID: clientID,
		Query:    listQuery,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list activity"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewActivityListResponseFromViews(views),
		int(total),
		listQuery,
	))
}
