package content

import (
	"context"
	"errors"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type GetContentStatsByClientQuery struct {
	ClientID uuid.UUID
}

type GetContentStatsByClientHandler struct {
	readRepo domaincontent.ContentReadRepository
}

func NewGetContentStatsByClientHandler(
	readRepo domaincontent.ContentReadRepository,
) *GetContentStatsByClientHandler {
	return &GetContentStatsByClientHandler{readRepo: readRepo}
}

func (h *GetContentStatsByClientHandler) Handle(
	ctx context.Context,
	q GetContentStatsByClientQuery,
) (*domaincontent.ContentStats, error) {
	if q.ClientID == uuid.Nil {
		return nil, errors.New("clientId is required")
	}
	stats, err := h.readRepo.CountStatsByClientID(ctx, q.ClientID)
	if err != nil {
		return nil, errors.New("failed to get content stats")
	}
	return stats, nil
}
