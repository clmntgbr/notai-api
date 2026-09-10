package client

import (
	"context"
	"errors"

	"go-api/internal/domain/paginate"
	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
)

type ListClientsByUserQuery struct {
	UserID uuid.UUID
	Query  paginate.PaginateQuery
}

type ListClientsByUserHandler struct {
	readRepo domainclient.ClientReadRepository
}

func NewListClientsByUserHandler(
	readRepo domainclient.ClientReadRepository,
) *ListClientsByUserHandler {
	return &ListClientsByUserHandler{readRepo: readRepo}
}

func (h *ListClientsByUserHandler) Handle(
	ctx context.Context,
	q ListClientsByUserQuery,
) ([]domainclient.ClientView, int64, error) {
	if q.UserID == uuid.Nil {
		return nil, 0, errors.New("userId is required")
	}
	views, total, err := h.readRepo.FindPageByUserID(ctx, q.UserID, q.Query)
	if err != nil {
		return nil, 0, errors.New("failed to list clients")
	}
	return views, total, nil
}
