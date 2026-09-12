package handler

import (
	"context"
	"io"

	querycontent "go-api/internal/application/query/content"
	domaincontent "go-api/internal/domain/content"
)

type contentGetByIDHandler interface {
	Handle(ctx context.Context, q querycontent.GetContentByIDQuery) (*domaincontent.ContentView, error)
}

type contentListByCampaignHandler interface {
	Handle(
		ctx context.Context,
		q querycontent.ListContentsByCampaignQuery,
	) (*querycontent.ListContentsByCampaignResult, error)
}

type contentStatsByClientHandler interface {
	Handle(ctx context.Context, q querycontent.GetContentStatsByClientQuery) (*domaincontent.ContentStats, error)
}

type contentStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}
