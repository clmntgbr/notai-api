package media

import (
	"context"
	"fmt"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetStatsByClientQuery struct {
	ClientID   uuid.UUID
	CampaignID *uuid.UUID // nil = all campaigns for the client
	// From/To define the inclusive stats window (UTC).
	From time.Time
	To   time.Time
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
	if q.From.IsZero() || q.To.IsZero() {
		return nil, fmt.Errorf("from and to are required")
	}
	if q.From.After(q.To) {
		return nil, fmt.Errorf("from must be before to")
	}
	stats, err := h.mediaRepo.CountStatsByClientID(ctx, q.ClientID, q.CampaignID, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("failed to get media stats: %w", err)
	}
	return stats, nil
}
