package handler

import (
	"errors"
	"io"
	"strconv"

	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type MediaHandler struct {
	presignHandler mediaPresignHandler
	listHandler    mediaListByCampaignHandler
	getByIDHandler mediaGetByIDHandler
	statsHandler   mediaStatsByClientHandler
	storage        mediaStorage
}

func NewMediaHandler(
	presignHandler mediaPresignHandler,
	listHandler mediaListByCampaignHandler,
	getByIDHandler mediaGetByIDHandler,
	statsHandler mediaStatsByClientHandler,
	storage mediaStorage,
) *MediaHandler {
	return &MediaHandler{
		presignHandler: presignHandler,
		listHandler:    listHandler,
		getByIDHandler: getByIDHandler,
		statsHandler:   statsHandler,
		storage:        storage,
	}
}

func (h *MediaHandler) Presign(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	var req dto.PresignMediaRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
	}

	files := make([]mediacmd.PresignFileInput, 0, len(req.Files))
	for _, file := range req.Files {
		files = append(files, mediacmd.PresignFileInput{
			Filename:    file.Filename,
			ContentType: file.ContentType,
		})
	}

	result, err := h.presignHandler.Handle(c.Context(), mediacmd.PresignMediaCommand{
		CampaignID: campaignID,
		ClientID:   clientID,
		Files:      files,
	})
	if err != nil {
		if errors.Is(err, domainmedia.ErrUnsupportedContentType) {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
				"message": "Unsupported media type",
			})
		}
		if errors.Is(err, domainmedia.ErrTooManyFiles) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "At most 20 files can be uploaded at once",
			})
		}
		if errors.Is(err, domainmedia.ErrEmptyFileList) || errors.Is(err, domainmedia.ErrInvalidFilename) {
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

	return c.Status(fiber.StatusCreated).JSON(presenter.NewPresignMediaResponse(result))
}

func (h *MediaHandler) ListByCampaign(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
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
		listQuery.SortBy = "created_at"
	}
	if orderBy == "" {
		listQuery.OrderBy = paginate.OrderByDesc
	}

	result, err := h.listHandler.Handle(c.Context(), querymedia.ListByCampaignQuery{
		ClientID:   clientID,
		CampaignID: campaignID,
		Query:      listQuery,
	})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list media"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewMediaListResponseFromViews(result.Views),
		int(result.Total),
		listQuery,
	))
}

func (h *MediaHandler) Stats(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	stats, err := h.statsHandler.Handle(c.Context(), querymedia.GetStatsByClientQuery{
		ClientID: clientID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get media stats"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaStatsResponse(stats))
}

func (h *MediaHandler) GetByID(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid media id"})
	}

	result, err := h.getByIDHandler.Handle(c.Context(), querymedia.GetByIDQuery{
		ID:       id,
		ClientID: clientID,
	})
	if err != nil {
		if err.Error() == "media not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get media"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewMediaDetailResponse(result))
}

func (h *MediaHandler) GetThumbnail(c fiber.Ctx) error {
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

	result, err := h.getByIDHandler.Handle(c.Context(), querymedia.GetByIDQuery{
		ID:       id,
		ClientID: clientID,
	})
	if err != nil {
		if err.Error() == "media not found" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	thumbKey := domainmedia.NewThumbnailKey(
		result.Media.ClientID,
		result.Media.CampaignID,
		result.Media.ID,
	)
	reader, err := h.storage.GetThumbnail(c.Context(), thumbKey)
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
	c.Set("ETag", strconv.Quote(strconv.FormatInt(result.Media.UpdatedAt.UnixNano(), 10)))
	c.Set("Content-Length", strconv.Itoa(len(body)))

	return c.Send(body)
}

func (h *MediaHandler) GetContentThumbnail(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	mediaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	contentID, err := uuid.Parse(c.Params("contentId"))
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}

	result, err := h.getByIDHandler.Handle(c.Context(), querymedia.GetByIDQuery{
		ID:       mediaID,
		ClientID: clientID,
	})
	if err != nil {
		if err.Error() == "media not found" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	var thumbKey string
	var updatedAt int64
	for _, child := range result.Contents {
		if child.ID != contentID {
			continue
		}
		if child.ThumbnailKey == nil || *child.ThumbnailKey == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		thumbKey = *child.ThumbnailKey
		updatedAt = child.UpdatedAt.UnixNano()
		break
	}
	if thumbKey == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}

	reader, err := h.storage.GetThumbnail(c.Context(), thumbKey)
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
	c.Set("ETag", strconv.Quote(strconv.FormatInt(updatedAt, 10)))
	c.Set("Content-Length", strconv.Itoa(len(body)))

	return c.Send(body)
}
