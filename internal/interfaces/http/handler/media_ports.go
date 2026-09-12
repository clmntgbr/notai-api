package handler

import (
	"context"
	"io"

	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
)

type mediaPresignHandler interface {
	Handle(ctx context.Context, cmd mediacmd.PresignMediaCommand) (*mediacmd.PresignMediaResult, error)
}

type mediaListByClientHandler interface {
	Handle(ctx context.Context, q querymedia.ListByClientQuery) (*querymedia.ListByClientResult, error)
}

type mediaGetByIDHandler interface {
	Handle(ctx context.Context, q querymedia.GetByIDQuery) (*querymedia.GetByIDResult, error)
}

type mediaStatsByClientHandler interface {
	Handle(ctx context.Context, q querymedia.GetStatsByClientQuery) (*domainmedia.MediaStats, error)
}

type mediaStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}
