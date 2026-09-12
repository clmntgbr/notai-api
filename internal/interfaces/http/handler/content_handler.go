package handler

import (
	"errors"
	"io"
	"strconv"

	contentcmd "go-api/internal/application/command/content"
	querycontent "go-api/internal/application/query/content"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ContentHandler struct {
	presignHandler contentPresignHandler
	getByIDHandler contentGetByIDHandler
	listHandler    contentListByCampaignHandler
	statsHandler   contentStatsByClientHandler
	storage        contentStorage
}

func NewContentHandler(
	presignHandler contentPresignHandler,
	getByIDHandler contentGetByIDHandler,
	listHandler contentListByCampaignHandler,
	statsHandler contentStatsByClientHandler,
	storage contentStorage,
) *ContentHandler {
	return &ContentHandler{
		presignHandler: presignHandler,
		getByIDHandler: getByIDHandler,
		listHandler:    listHandler,
		statsHandler:   statsHandler,
		storage:        storage,
	}
}

func (h *ContentHandler) Presign(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	var req dto.PresignContentsRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
	}

	var campaignID uuid.UUID
	if req.CampaignID != "" {
		campaignID, err = uuid.Parse(req.CampaignID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
		}
	}

	files := make([]contentcmd.PresignFileInput, 0, len(req.Files))
	for _, file := range req.Files {
		files = append(files, contentcmd.PresignFileInput{
			Filename:    file.Filename,
			ContentType: file.ContentType,
		})
	}

	result, err := h.presignHandler.Handle(c.Context(), contentcmd.PresignContentsCommand{
		CampaignID: campaignID,
		ClientID:   clientID,
		Files:      files,
	})
	if err != nil {
		if errors.Is(err, domaincontent.ErrUnsupportedContentType) {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
				"message": "Unsupported media type",
			})
		}
		if errors.Is(err, domaincontent.ErrTooManyFiles) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "At most 20 files can be uploaded at once",
			})
		}
		if errors.Is(err, domaincontent.ErrEmptyFileList) || errors.Is(err, domaincontent.ErrInvalidFilename) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Invalid files payload",
			})
		}
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate upload urls",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(presenter.NewPresignContentsResponse(result))
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
