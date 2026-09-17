package quota

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type QuotaWriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, quota *Quota) error
	Update(ctx context.Context, quota *Quota) error
	GetByID(ctx context.Context, id uuid.UUID) (*Quota, error)
}

type QuotaReadRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*QuotaView, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]QuotaView, error)
}

type QuotaView struct {
	ID   uuid.UUID
	Name string

	MaxClientMembers         int
	MaxCampaigns             int
	MaxVerificationsPerMonth int
	MaxConcurrentAnalyses    int
	MaxFileSizeMB            int
	ReportRetentionDays      int
	AllowsVideoAnalysis      bool
	AllowsPDFExport          bool
	AllowsCSVExport          bool
	AllowsAPIAccess          bool
	OveragePriceCents        int
	AnalysisPriority         int

	MaxDetectorsPerAnalysis        int
	MaxFramesPerVideo              int
	AllowsReanalysis               bool
	MaxStorageGB                   int
	MaxBatchUploadSize             int
	FrameRetentionDays             int
	AllowsCustomRuleset            bool
	AllowsWhiteLabelReport         bool
	AllowsWebhooks                 bool
	QuotaOverageGraceVerifications int

	CreatedAt time.Time
	UpdatedAt time.Time
}
