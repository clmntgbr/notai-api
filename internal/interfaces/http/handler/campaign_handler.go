package handler

import (
	"errors"
	"io"
	"strconv"
	"strings"

	campaigncmd "go-api/internal/application/command/campaign"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/paginate"
	httpctx "go-api/internal/interfaces/http/context"
	"go-api/internal/interfaces/http/dto"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/validation"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type CampaignHandler struct {
	createHandler            campaignCreateHandler
	updateHandler            campaignUpdateHandler
	deleteHandler            campaignDeleteHandler
	getByIDHandler           campaignGetByIDHandler
	listByClientHandler      campaignListByClientHandler
	getClientByIDHandler     campaignGetClientByIDHandler
	presignBackgroundHandler campaignPresignBackgroundHandler
	clearBackgroundHandler   campaignClearBackgroundHandler
	storage                  campaignStorage
}

func NewCampaignHandler(
	createHandler campaignCreateHandler,
	updateHandler campaignUpdateHandler,
	deleteHandler campaignDeleteHandler,
	getByIDHandler campaignGetByIDHandler,
	listByClientHandler campaignListByClientHandler,
	getClientByIDHandler campaignGetClientByIDHandler,
	presignBackgroundHandler campaignPresignBackgroundHandler,
	clearBackgroundHandler campaignClearBackgroundHandler,
	storage campaignStorage,
) *CampaignHandler {
	return &CampaignHandler{
		createHandler:            createHandler,
		updateHandler:            updateHandler,
		deleteHandler:            deleteHandler,
		getByIDHandler:           getByIDHandler,
		listByClientHandler:      listByClientHandler,
		getClientByIDHandler:     getClientByIDHandler,
		presignBackgroundHandler: presignBackgroundHandler,
		clearBackgroundHandler:   clearBackgroundHandler,
		storage:                  storage,
	}
}

func (h *CampaignHandler) Create(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	var req dto.CreateCampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	campaign, err := h.createHandler.Handle(c.Context(), campaigncmd.CreateCampaignCommand{
		Name:      req.Name,
		ClientID:  clientID,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
	})
	if err != nil {
		if errors.Is(err, domaincampaign.ErrInvalidSchedule) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "endAt must be after startAt"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to create campaign"})
	}

	return c.Status(fiber.StatusCreated).JSON(presenter.NewCampaignDetailResponseFromEntity(*campaign))
}

func (h *CampaignHandler) GetByID(c fiber.Ctx) error {
	user, err := httpctx.GetUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	view, err := h.getByIDHandler.Handle(c.Context(), querycampaign.GetCampaignByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get campaign"})
	}

	if view.ClientID == clientID {
		return c.Status(fiber.StatusOK).JSON(presenter.NewCampaignDetailResponseFromView(*view))
	}

	ownerClient, err := h.getClientByIDHandler.Handle(c.Context(), queryclient.GetClientByIDQuery{ID: view.ClientID})
	if err != nil || ownerClient == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
	}

	isMember := false
	for _, memberID := range ownerClient.MemberIDs {
		if memberID == user.ID {
			isMember = true
			break
		}
	}
	if !isMember {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
	}

	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"code":       "WRONG_CLIENT",
		"message":    "Campaign belongs to another client",
		"clientId":   ownerClient.ID.String(),
		"clientName": ownerClient.Name,
	})
}

func (h *CampaignHandler) List(c fiber.Ctx) error {
	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	var query paginate.PaginateQuery
	if err := c.Bind().Query(&query); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid query parameters",
		})
	}

	orderBy := query.OrderBy
	sortBy := query.SortBy
	query.Normalize()
	if sortBy == "" {
		query.SortBy = "created_at"
	}
	if orderBy == "" {
		query.OrderBy = paginate.OrderByDesc
	}

	views, total, err := h.listByClientHandler.Handle(c.Context(), querycampaign.ListCampaignsByClientQuery{
		ClientID: clientID,
		Query:    query,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to list campaigns"})
	}

	return c.Status(fiber.StatusOK).JSON(paginate.NewPaginateResponse(
		presenter.NewCampaignListResponseFromViews(views),
		int(total),
		query,
	))
}

func (h *CampaignHandler) Update(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	existing, err := h.getByIDHandler.Handle(c.Context(), querycampaign.GetCampaignByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get campaign"})
	}
	if existing.ClientID != clientID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
	}

	var req dto.UpdateCampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid request body"})
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validation.Struct(c, &req); err != nil {
		return err
	}

	err = h.updateHandler.Handle(c.Context(), campaigncmd.UpdateCampaignCommand{
		ID:        id,
		Name:      req.Name,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
	})
	if err != nil {
		if errors.Is(err, domaincampaign.ErrDefaultCampaignProtected) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Default campaign cannot be modified",
				"code":    "DEFAULT_CAMPAIGN_PROTECTED",
			})
		}
		if errors.Is(err, domaincampaign.ErrInvalidSchedule) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "endAt must be after startAt"})
		}
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to update campaign"})
	}

	view, err := h.getByIDHandler.Handle(c.Context(), querycampaign.GetCampaignByIDQuery{ID: id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get campaign"})
	}

	return c.Status(fiber.StatusOK).JSON(presenter.NewCampaignDetailResponseFromView(*view))
}

func (h *CampaignHandler) Delete(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	existing, err := h.getByIDHandler.Handle(c.Context(), querycampaign.GetCampaignByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to get campaign"})
	}
	if existing.ClientID != clientID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
	}

	if err := h.deleteHandler.Handle(c.Context(), campaigncmd.DeleteCampaignCommand{ID: id}); err != nil {
		if errors.Is(err, domaincampaign.ErrDefaultCampaignProtected) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Default campaign cannot be modified",
				"code":    "DEFAULT_CAMPAIGN_PROTECTED",
			})
		}
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to delete campaign"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CampaignHandler) PresignBackground(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	var req dto.PresignCampaignBackgroundRequest
	if err := validation.BindBody(c, &req); err != nil {
		return err
	}

	result, err := h.presignBackgroundHandler.Handle(c.Context(), campaigncmd.PresignBackgroundCommand{
		CampaignID:  id,
		ClientID:    clientID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
	})
	if err != nil {
		if errors.Is(err, domaincampaign.ErrUnsupportedBackgroundType) {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
				"message": "Unsupported media type",
			})
		}
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate upload url",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.PresignCampaignBackgroundResponse{URL: result.URL})
}

func (h *CampaignHandler) GetThumbnail(c fiber.Ctx) error {
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

	view, err := h.getByIDHandler.Handle(c.Context(), querycampaign.GetCampaignByIDQuery{ID: id})
	if err != nil {
		if err.Error() == "campaign not found" {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if view.ClientID != clientID {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if view.BackgroundThumbnailKey == "" {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if view.BackgroundStatus != domaincampaign.BackgroundStatusReady &&
		view.BackgroundStatus != domaincampaign.BackgroundStatusPending {
		return c.SendStatus(fiber.StatusNotFound)
	}

	reader, err := h.storage.GetThumbnail(c.Context(), view.BackgroundThumbnailKey)
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

func (h *CampaignHandler) ClearBackground(c fiber.Ctx) error {
	if _, err := httpctx.GetUser(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	clientID, err := httpctx.GetCurrentClientID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Current client is required"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid campaign id"})
	}

	if err := h.clearBackgroundHandler.Handle(c.Context(), campaigncmd.ClearBackgroundCommand{
		CampaignID: id,
		ClientID:   clientID,
	}); err != nil {
		if err.Error() == "campaign not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Campaign not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to clear background",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
