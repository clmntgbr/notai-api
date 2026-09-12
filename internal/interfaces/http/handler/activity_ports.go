package handler

import (
	"context"

	queryactivity "go-api/internal/application/query/activity"
	domainactivity "go-api/internal/domain/activity"
)

type activityListByClientHandler interface {
	Handle(ctx context.Context, q queryactivity.ListByClientQuery) ([]domainactivity.View, int64, error)
}
