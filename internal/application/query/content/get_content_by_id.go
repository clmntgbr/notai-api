package content

import (
	"context"
	"errors"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type GetContentByIDQuery struct {
	ID uuid.UUID
}

type GetContentByIDHandler struct {
	readRepo domaincontent.ContentReadRepository
}

func NewGetContentByIDHandler(readRepo domaincontent.ContentReadRepository) *GetContentByIDHandler {
	return &GetContentByIDHandler{readRepo: readRepo}
}

func (h *GetContentByIDHandler) Handle(
	ctx context.Context,
	q GetContentByIDQuery,
) (*domaincontent.ContentView, error) {
	view, err := h.readRepo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, errors.New("failed to get content")
	}
	if view == nil {
		return nil, errors.New("content not found")
	}
	return view, nil
}
