package handler

import (
	"errors"
	"io"
	"strconv"

	contentcmd "go-api/internal/application/command/content"
	querycontent "go-api/internal/application/query/content"
	domaincontent "go-api/internal/domain/content"
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
	storage        contentStorage
}

func NewContentHandler(
	presignHandler contentPresignHandler,
	getByIDHandler contentGetByIDHandler,
	storage contentStorage,
) *ContentHandler {
	return &ContentHandler{
		presignHandler: presignHandler,
		getByIDHandler: getByIDHandler,
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

	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	var req dto.PresignContentsRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
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
