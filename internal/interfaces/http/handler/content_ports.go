package handler

import (
	"context"
	"io"

	contentcmd "go-api/internal/application/command/content"
	querycontent "go-api/internal/application/query/content"
	domaincontent "go-api/internal/domain/content"
)

type contentPresignHandler interface {
	Handle(ctx context.Context, cmd contentcmd.PresignContentsCommand) (*contentcmd.PresignContentsResult, error)
}

type contentGetByIDHandler interface {
	Handle(ctx context.Context, q querycontent.GetContentByIDQuery) (*domaincontent.ContentView, error)
}

type contentListByCampaignHandler interface {
	Handle(
		ctx context.Context,
		q querycontent.ListContentsByCampaignQuery,
	) ([]domaincontent.ContentView, int64, error)
}

type contentStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}
