package quota

import (
	"time"

	"github.com/google/uuid"
)

type Quota struct {
	ID   uuid.UUID
	Name string

	MaxClientMembers          int
	MaxCampaigns              int
	MaxVerificationsPerMonth  int
	MaxConcurrentAnalyses     int
	MaxFileSizeMB             int
	ReportRetentionDays       int
	AllowsVideoAnalysis       bool
	AllowsPDFExport           bool
	AllowsCSVExport           bool
	AllowsAPIAccess           bool
	OveragePriceCents         int
	AnalysisPriority          int

	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewQuotaParams struct {
	Name                     string
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
}

func NewQuota(p NewQuotaParams) *Quota {
	now := time.Now().UTC()
	return &Quota{
		ID:                       uuid.New(),
		Name:                     p.Name,
		MaxClientMembers:         p.MaxClientMembers,
		MaxCampaigns:             p.MaxCampaigns,
		MaxVerificationsPerMonth: p.MaxVerificationsPerMonth,
		MaxConcurrentAnalyses:    p.MaxConcurrentAnalyses,
		MaxFileSizeMB:            p.MaxFileSizeMB,
		ReportRetentionDays:      p.ReportRetentionDays,
		AllowsVideoAnalysis:      p.AllowsVideoAnalysis,
		AllowsPDFExport:          p.AllowsPDFExport,
		AllowsCSVExport:          p.AllowsCSVExport,
		AllowsAPIAccess:          p.AllowsAPIAccess,
		OveragePriceCents:        p.OveragePriceCents,
		AnalysisPriority:         p.AnalysisPriority,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

func (q *Quota) ApplyUpdate(p NewQuotaParams) {
	q.Name = p.Name
	q.MaxClientMembers = p.MaxClientMembers
	q.MaxCampaigns = p.MaxCampaigns
	q.MaxVerificationsPerMonth = p.MaxVerificationsPerMonth
	q.MaxConcurrentAnalyses = p.MaxConcurrentAnalyses
	q.MaxFileSizeMB = p.MaxFileSizeMB
	q.ReportRetentionDays = p.ReportRetentionDays
	q.AllowsVideoAnalysis = p.AllowsVideoAnalysis
	q.AllowsPDFExport = p.AllowsPDFExport
	q.AllowsCSVExport = p.AllowsCSVExport
	q.AllowsAPIAccess = p.AllowsAPIAccess
	q.OveragePriceCents = p.OveragePriceCents
	q.AnalysisPriority = p.AnalysisPriority
	q.UpdatedAt = time.Now().UTC()
}
