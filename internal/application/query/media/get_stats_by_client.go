package media

import (
	"context"
	"fmt"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetStatsByClientQuery struct {
	ClientID uuid.UUID
}

type GetStatsByClientHandler struct {
	mediaRepo domainmedia.MediaReadRepository
}

func NewGetStatsByClientHandler(mediaRepo domainmedia.MediaReadRepository) *GetStatsByClientHandler {
	return &GetStatsByClientHandler{mediaRepo: mediaRepo}
}

func (h *GetStatsByClientHandler) Handle(
	ctx context.Context,
	q GetStatsByClientQuery,
) (*domainmedia.MediaStats, error) {
	if q.ClientID == uuid.Nil {
		return nil, fmt.Errorf("clientId is required")
	}
	stats, err := h.mediaRepo.CountStatsByClientID(ctx, q.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media stats: %w", err)
	}
	return stats, nil
}
