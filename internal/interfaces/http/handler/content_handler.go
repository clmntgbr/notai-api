package handler

import (
	"io"
	"strconv"

	querycontent "go-api/internal/application/query/content"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/presenter"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ContentHandler struct {
	getByIDHandler contentGetByIDHandler
	listHandler    contentListByCampaignHandler
	statsHandler   contentStatsByClientHandler
	storage        contentStorage
}

func NewContentHandler(
	getByIDHandler contentGetByIDHandler,
	listHandler contentListByCampaignHandler,
	statsHandler contentStatsByClientHandler,
	storage contentStorage,
) *ContentHandler {
	return &ContentHandler{
		getByIDHandler: getByIDHandler,
		listHandler:    listHandler,
		statsHandler:   statsHandler,
		storage:        storage,
	}
}

func (h *ContentHandler) List(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	var listQuery struct {
		paginate.PaginateQuery
		CampaignID string `query:"campaignId"`
	}
	if err := c.Bind().Query(&listQuery); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	var campaignID uuid.UUID
	if listQuery.CampaignID != "" {
		campaignID, err = uuid.Parse(listQuery.CampaignID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
		}
	}

	sortBy, orderBy := listQuery.SortBy, listQuery.OrderBy
	listQuery.Normalize()
	if sortBy == "" {
		listQuery.SortBy = "created_at"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByDesc
	}

	result, err := h.listHandler.Handle(c.Context(), querycontent.ListContentsByCampaignQuery{
		CampaignID: campaignID,
		ClientID:   clientID,
		Query:      listQuery.PaginateQuery,
	})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list contents"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewContentListResponseFromViews(result.Views, result.Campaigns),
		int(result.Total),
		listQuery.PaginateQuery,
	))
}

func (h *ContentHandler) Stats(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	stats, err := h.statsHandler.Handle(c.Context(), querycontent.GetContentStatsByClientQuery{
		ClientID: clientID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get content stats"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewContentStatsResponse(stats))
}

func (h *ContentHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid content id"})
	}

	view, err := h.getByIDHandler.Handle(c.Context(), querycontent.GetContentByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "content not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Content not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get content"})
	}
	if view.ClientID != clientID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Content not found"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewContentDetailResponseFromView(*view))
}

func (h *ContentHandler) GetThumbnail(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	view, err := h.getByIDHandler.Handle(c.Context(), querycontent.GetContentByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "content not found" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if view.ClientID != clientID {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if view.ThumbnailKey == nil || *view.ThumbnailKey == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}

	reader, err := h.storage.GetThumbnail(c.Context(), *view.ThumbnailKey)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Set("Content-Type", "image/jpeg")
	c.Set("Cache-Control", "private, no-cache")
	c.Set("ETag", strconv.Quote(strconv.FormatInt(view.UpdatedAt.UnixNano(), 10)))
	c.Set("Content-Length", strconv.Itoa(len(body)))

	return c.Send(body)
}
