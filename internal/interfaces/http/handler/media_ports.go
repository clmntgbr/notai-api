package handler

import (
	"context"
	"io"

	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
)

type mediaPresignHandler interface {
	Handle(ctx context.Context, cmd mediacmd.PresignMediaCommand) (*mediacmd.PresignMediaResult, error)
}

type mediaListByCampaignHandler interface {
	Handle(ctx context.Context, q querymedia.ListByCampaignQuery) (*querymedia.ListByCampaignResult, error)
}

type mediaGetByIDHandler interface {
	Handle(ctx context.Context, q querymedia.GetByIDQuery) (*querymedia.GetByIDResult, error)
}

type mediaStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}
