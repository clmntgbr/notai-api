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

	var listQuery struct {
		paginate.PaginateQuery
		CampaignIDs string `query:"campaignIds"`
	}
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	campaignIDs, err := parseUUIDFilters(listQuery.CampaignIDs)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid campaign ids",
			"errors":  fiber.Map{"campaignIds": err.Error()},
		})
	}

	from, to, err := parseOptionalMediaPeriod(c.Query("from"), c.Query("to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": err.Error(),
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
		ClientID:    clientID,
		CampaignIDs: campaignIDs,
		From:        from,
		To:          to,
		Query:       listQuery.PaginateQuery,
	})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list activity"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewActivityListResponseFromViews(views),
		int(total),
		listQuery.PaginateQuery,
	))
}
