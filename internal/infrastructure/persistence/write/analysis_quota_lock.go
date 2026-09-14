package write

import (
	"context"
	"errors"
	"hash/fnv"

	"go-api/internal/domain/port"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type analysisQuotaLocker struct {
	db *gorm.DB
}

func NewAnalysisQuotaLocker(db *gorm.DB) port.AnalysisQuotaLocker {
	return &analysisQuotaLocker{db: db}
}

func (l *analysisQuotaLocker) Lock(ctx context.Context, workspaceID uuid.UUID) error {
	if workspaceID == uuid.Nil {
		return errors.New("workspace id is required for analysis quota lock")
	}
	// Transaction-scoped lock: blocks other analysis starts for this workspace until commit.
	return DBWithContext(ctx, l.db).Exec(
		`SELECT pg_advisory_xact_lock(?)`,
		analysisQuotaLockKey(workspaceID),
	).Error
}

func analysisQuotaLockKey(workspaceID uuid.UUID) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("analysis_quota:"))
	_, _ = h.Write(workspaceID[:])
	return int64(h.Sum64())
}
