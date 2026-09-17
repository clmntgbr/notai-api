package subscription

import (
	"time"

	"github.com/google/uuid"
)

type QuotaCounter struct {
	Used int64
	Max  int
	Left int64
}

type MonthlyQuotaCounter struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	Used        int64
	Max         int
	Left        int64
}

type QuotaLimits struct {
	MaxFileSizeMB                  int
	ReportRetentionDays            int
	AllowsVideoAnalysis            bool
	AllowsPDFExport                bool
	AllowsCSVExport                bool
	AllowsAPIAccess                bool
	OveragePriceCents              int
	AnalysisPriority               int
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
}

type QuotaUsageView struct {
	WorkspaceID        uuid.UUID
	Members            QuotaCounter
	Campaigns          QuotaCounter
	Verifications      MonthlyQuotaCounter
	ConcurrentAnalyses QuotaCounter
	// Storage uses bytes for Used/Max/Left; Limits.MaxStorageGB remains the plan field.
	Storage     QuotaCounter
	Limits      QuotaLimits
	PeriodStart time.Time
	PeriodEnd   time.Time
}
