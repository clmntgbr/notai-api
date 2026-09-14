package handler

import (
	"context"

	queryworkspace "go-api/internal/application/query/workspace"
	domainworkspace "go-api/internal/domain/workspace"
)

type workspaceGetOwnedHandler interface {
	Handle(ctx context.Context, q queryworkspace.GetOwnedWorkspaceQuery) (*domainworkspace.WorkspaceView, error)
}
