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
	MaxFileSizeMB       int
	ReportRetentionDays int
	AllowsVideoAnalysis bool
	AllowsPDFExport     bool
	AllowsCSVExport     bool
	AllowsAPIAccess     bool
	OveragePriceCents   int
	AnalysisPriority    int
}

type QuotaUsageView struct {
	WorkspaceID         uuid.UUID
	Members             QuotaCounter
	Campaigns           QuotaCounter
	Verifications       MonthlyQuotaCounter
	ConcurrentAnalyses  QuotaCounter
	Limits              QuotaLimits
	PeriodStart         time.Time
	PeriodEnd           time.Time
}
