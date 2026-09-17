package read

import (
	"context"
	"errors"
	"time"

	domainquota "go-api/internal/domain/quota"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type quotaRow struct {
	ID                             uuid.UUID `gorm:"column:id"`
	Name                           string    `gorm:"column:name"`
	MaxClientMembers               int       `gorm:"column:max_client_members"`
	MaxCampaigns                   int       `gorm:"column:max_campaigns"`
	MaxVerificationsPerMonth       int       `gorm:"column:max_verifications_per_month"`
	MaxConcurrentAnalyses          int       `gorm:"column:max_concurrent_analyses"`
	MaxFileSizeMB                  int       `gorm:"column:max_file_size_mb"`
	ReportRetentionDays            int       `gorm:"column:report_retention_days"`
	AllowsVideoAnalysis            bool      `gorm:"column:allows_video_analysis"`
	AllowsPDFExport                bool      `gorm:"column:allows_pdf_export"`
	AllowsCSVExport                bool      `gorm:"column:allows_csv_export"`
	AllowsAPIAccess                bool      `gorm:"column:allows_api_access"`
	OveragePriceCents              int       `gorm:"column:overage_price_cents"`
	AnalysisPriority               int       `gorm:"column:analysis_priority"`
	MaxDetectorsPerAnalysis        int       `gorm:"column:max_detectors_per_analysis"`
	MaxFramesPerVideo              int       `gorm:"column:max_frames_per_video"`
	AllowsReanalysis               bool      `gorm:"column:allows_reanalysis"`
	MaxStorageGB                   int       `gorm:"column:max_storage_gb"`
	MaxBatchUploadSize             int       `gorm:"column:max_batch_upload_size"`
	FrameRetentionDays             int       `gorm:"column:frame_retention_days"`
	AllowsCustomRuleset            bool      `gorm:"column:allows_custom_ruleset"`
	AllowsWhiteLabelReport         bool      `gorm:"column:allows_white_label_report"`
	AllowsWebhooks                 bool      `gorm:"column:allows_webhooks"`
	QuotaOverageGraceVerifications int       `gorm:"column:quota_overage_grace_verifications"`
	CreatedAt                      time.Time `gorm:"column:created_at"`
	UpdatedAt                      time.Time `gorm:"column:updated_at"`
}

func (quotaRow) TableName() string { return "quotas" }

type quotaReadRepository struct {
	db *gorm.DB
}

func NewQuotaReadRepository(db *gorm.DB) domainquota.QuotaReadRepository {
	return &quotaReadRepository{db: db}
}

func (r *quotaReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainquota.QuotaView, error) {
	var row quotaRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	view := toQuotaView(row)
	return &view, nil
}

func (r *quotaReadRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]domainquota.QuotaView, error) {
	if len(ids) == 0 {
		return []domainquota.QuotaView{}, nil
	}
	var rows []quotaRow
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainquota.QuotaView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toQuotaView(row))
	}
	return out, nil
}

func toQuotaView(row quotaRow) domainquota.QuotaView {
	return domainquota.QuotaView{
		ID:                             row.ID,
		Name:                           row.Name,
		MaxClientMembers:               row.MaxClientMembers,
		MaxCampaigns:                   row.MaxCampaigns,
		MaxVerificationsPerMonth:       row.MaxVerificationsPerMonth,
		MaxConcurrentAnalyses:          row.MaxConcurrentAnalyses,
		MaxFileSizeMB:                  row.MaxFileSizeMB,
		ReportRetentionDays:            row.ReportRetentionDays,
		AllowsVideoAnalysis:            row.AllowsVideoAnalysis,
		AllowsPDFExport:                row.AllowsPDFExport,
		AllowsCSVExport:                row.AllowsCSVExport,
		AllowsAPIAccess:                row.AllowsAPIAccess,
		OveragePriceCents:              row.OveragePriceCents,
		AnalysisPriority:               row.AnalysisPriority,
		MaxDetectorsPerAnalysis:        row.MaxDetectorsPerAnalysis,
		MaxFramesPerVideo:              row.MaxFramesPerVideo,
		AllowsReanalysis:               row.AllowsReanalysis,
		MaxStorageGB:                   row.MaxStorageGB,
		MaxBatchUploadSize:             row.MaxBatchUploadSize,
		FrameRetentionDays:             row.FrameRetentionDays,
		AllowsCustomRuleset:            row.AllowsCustomRuleset,
		AllowsWhiteLabelReport:         row.AllowsWhiteLabelReport,
		AllowsWebhooks:                 row.AllowsWebhooks,
		QuotaOverageGraceVerifications: row.QuotaOverageGraceVerifications,
		CreatedAt:                      row.CreatedAt,
		UpdatedAt:                      row.UpdatedAt,
	}
}
