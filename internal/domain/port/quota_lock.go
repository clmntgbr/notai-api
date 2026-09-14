package port

import (
	"context"

	"github.com/google/uuid"
)

// AnalysisQuotaLocker serializes analysis-slot checks per workspace.
// Lock must be called inside an open DB transaction; it is released on commit/rollback.
type AnalysisQuotaLocker interface {
	Lock(ctx context.Context, workspaceID uuid.UUID) error
}
