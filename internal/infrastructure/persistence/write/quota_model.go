package write

import (
	"time"

	domainquota "go-api/internal/domain/quota"

	"github.com/google/uuid"
)

type QuotaModel struct {
	ID   uuid.UUID `gorm:"column:id;primaryKey"`
	Name string    `gorm:"column:name"`

	MaxClientMembers         int  `gorm:"column:max_client_members"`
	MaxCampaigns             int  `gorm:"column:max_campaigns"`
	MaxVerificationsPerMonth int  `gorm:"column:max_verifications_per_month"`
	MaxConcurrentAnalyses    int  `gorm:"column:max_concurrent_analyses"`
	MaxFileSizeMB            int  `gorm:"column:max_file_size_mb"`
	ReportRetentionDays      int  `gorm:"column:report_retention_days"`
	AllowsVideoAnalysis      bool `gorm:"column:allows_video_analysis"`
	AllowsPDFExport          bool `gorm:"column:allows_pdf_export"`
	AllowsCSVExport          bool `gorm:"column:allows_csv_export"`
	AllowsAPIAccess          bool `gorm:"column:allows_api_access"`
	OveragePriceCents        int  `gorm:"column:overage_price_cents"`
	AnalysisPriority         int  `gorm:"column:analysis_priority"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (QuotaModel) TableName() string { return "quotas" }

func quotaModelFromDomain(q *domainquota.Quota) *QuotaModel {
	return &QuotaModel{
		ID:                       q.ID,
		Name:                     q.Name,
		MaxClientMembers:         q.MaxClientMembers,
		MaxCampaigns:             q.MaxCampaigns,
		MaxVerificationsPerMonth: q.MaxVerificationsPerMonth,
		MaxConcurrentAnalyses:    q.MaxConcurrentAnalyses,
		MaxFileSizeMB:            q.MaxFileSizeMB,
		ReportRetentionDays:      q.ReportRetentionDays,
		AllowsVideoAnalysis:      q.AllowsVideoAnalysis,
		AllowsPDFExport:          q.AllowsPDFExport,
		AllowsCSVExport:          q.AllowsCSVExport,
		AllowsAPIAccess:          q.AllowsAPIAccess,
		OveragePriceCents:        q.OveragePriceCents,
		AnalysisPriority:         q.AnalysisPriority,
		CreatedAt:                q.CreatedAt,
		UpdatedAt:                q.UpdatedAt,
	}
}

func quotaDomainFromModel(m *QuotaModel) *domainquota.Quota {
	return &domainquota.Quota{
		ID:                       m.ID,
		Name:                     m.Name,
		MaxClientMembers:         m.MaxClientMembers,
		MaxCampaigns:             m.MaxCampaigns,
		MaxVerificationsPerMonth: m.MaxVerificationsPerMonth,
		MaxConcurrentAnalyses:    m.MaxConcurrentAnalyses,
		MaxFileSizeMB:            m.MaxFileSizeMB,
		ReportRetentionDays:      m.ReportRetentionDays,
		AllowsVideoAnalysis:      m.AllowsVideoAnalysis,
		AllowsPDFExport:          m.AllowsPDFExport,
		AllowsCSVExport:          m.AllowsCSVExport,
		AllowsAPIAccess:          m.AllowsAPIAccess,
		OveragePriceCents:        m.OveragePriceCents,
		AnalysisPriority:         m.AnalysisPriority,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}
