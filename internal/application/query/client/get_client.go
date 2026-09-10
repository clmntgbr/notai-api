package client

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
)

type GetClientByIDQuery struct {
	ID uuid.UUID
}

type GetClientByIDHandler struct {
	readRepo domainclient.ClientReadRepository
}

func NewGetClientByIDHandler(
	readRepo domainclient.ClientReadRepository,
) *GetClientByIDHandler {
	return &GetClientByIDHandler{readRepo: readRepo}
}

func (h *GetClientByIDHandler) Handle(
	ctx context.Context,
	q GetClientByIDQuery,
) (*domainclient.ClientView, error) {
	view, err := h.readRepo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, errors.New("failed to get client")
	}
	if view == nil {
		return nil, errors.New("client not found")
	}
	return view, nil
}
