package workspace

import (
	"context"
	"errors"

	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

var ErrWorkspaceNotFound = errors.New("workspace not found")

type GetOwnedWorkspaceQuery struct {
	OwnerUserID uuid.UUID
}

type GetOwnedWorkspaceHandler struct {
	workspaceRepo domainworkspace.WorkspaceReadRepository
}

func NewGetOwnedWorkspaceHandler(
	workspaceRepo domainworkspace.WorkspaceReadRepository,
) *GetOwnedWorkspaceHandler {
	return &GetOwnedWorkspaceHandler{workspaceRepo: workspaceRepo}
}

func (h *GetOwnedWorkspaceHandler) Handle(
	ctx context.Context,
	q GetOwnedWorkspaceQuery,
) (*domainworkspace.WorkspaceView, error) {
	view, err := h.workspaceRepo.FindByOwnerUserID(ctx, q.OwnerUserID)
	if err != nil {
		return nil, errors.New("failed to get workspace")
	}
	if view == nil {
		return nil, ErrWorkspaceNotFound
	}
	return view, nil
}
