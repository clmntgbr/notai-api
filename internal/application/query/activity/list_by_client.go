package activity

import (
	"context"
	"errors"

	domainactivity "go-api/internal/domain/activity"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListByClientQuery struct {
	ClientID uuid.UUID
	Query    paginate.PaginateQuery
}

type ListByClientHandler struct {
	repo domainactivity.ReadRepository
}

func NewListByClientHandler(repo domainactivity.ReadRepository) *ListByClientHandler {
	return &ListByClientHandler{repo: repo}
}

func (h *ListByClientHandler) Handle(
	ctx context.Context,
	q ListByClientQuery,
) ([]domainactivity.View, int64, error) {
	if q.ClientID == uuid.Nil {
		return nil, 0, errors.New("clientId is required")
	}
	views, total, err := h.repo.FindPageByClientID(ctx, q.ClientID, q.Query)
	if err != nil {
		return nil, 0, errors.New("failed to list activity")
	}
	return views, total, nil
}
