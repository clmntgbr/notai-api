package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetByIDQuery struct {
	ID       uuid.UUID
	ClientID uuid.UUID
}

type GetByIDResult struct {
	Media    domainmedia.MediaView
	Contents []domainmedia.ContentChildView
}

type GetByIDHandler struct {
	mediaRepo domainmedia.MediaReadRepository
}

func NewGetByIDHandler(mediaRepo domainmedia.MediaReadRepository) *GetByIDHandler {
	return &GetByIDHandler{mediaRepo: mediaRepo}
}

func (h *GetByIDHandler) Handle(ctx context.Context, q GetByIDQuery) (*GetByIDResult, error) {
	if q.ID == uuid.Nil {
		return nil, errors.New("media not found")
	}
	view, err := h.mediaRepo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if view == nil || (q.ClientID != uuid.Nil && view.ClientID != q.ClientID) {
		return nil, errors.New("media not found")
	}

	contents, err := h.mediaRepo.FindContentsByMediaID(ctx, view.ID)
	if err != nil {
		return nil, err
	}
	return &GetByIDResult{Media: *view, Contents: contents}, nil
}
