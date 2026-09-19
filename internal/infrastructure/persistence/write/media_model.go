package write

import (
	"encoding/json"
	"strings"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
)

type MediaModel struct {
	ID            uuid.UUID    `gorm:"column:id;primaryKey"`
	CampaignID    uuid.UUID    `gorm:"column:campaign_id"`
	ClientID      uuid.UUID    `gorm:"column:client_id"`
	Filename      string       `gorm:"column:filename"`
	ContentType   string       `gorm:"column:content_type"`
	MediaType     string       `gorm:"column:media_type"`
	ObjectKey     string       `gorm:"column:object_key"`
	SizeBytes     *int64       `gorm:"column:size_bytes"`
	Status        string       `gorm:"column:status"`
	Verdict       dbtype.JSONB `gorm:"column:verdict"`
	FailureReason *string      `gorm:"column:failure_reason"`
	AnalyzedAt    *time.Time   `gorm:"column:analyzed_at"`
	CreatedAt     time.Time    `gorm:"column:created_at"`
	UpdatedAt     time.Time    `gorm:"column:updated_at"`
	DeletedAt     *time.Time   `gorm:"column:deleted_at"`
}

func (MediaModel) TableName() string { return "media" }

func mediaModelFromDomain(m *domainmedia.Media) *MediaModel {
	var verdict dbtype.JSONB
	if m.Verdict != nil {
		raw, _ := json.Marshal(m.Verdict)
		verdict = dbtype.JSONB(raw)
	}
	var failureReason *string
	if reason := strings.TrimSpace(m.FailureReason); reason != "" {
		failureReason = &reason
	}
	return &MediaModel{
		ID:            m.ID,
		CampaignID:    m.CampaignID,
		ClientID:      m.ClientID,
		Filename:      m.Filename,
		ContentType:   m.ContentType,
		MediaType:     string(m.MediaType),
		ObjectKey:     m.ObjectKey,
		SizeBytes:     m.SizeBytes,
		Status:        string(m.Status),
		Verdict:       verdict,
		FailureReason: failureReason,
		AnalyzedAt:    m.AnalyzedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
}

func mediaDomainFromModel(m *MediaModel) *domainmedia.Media {
	var verdict *domainmedia.Verdict
	if len(m.Verdict) > 0 {
		var v domainmedia.Verdict
		if err := json.Unmarshal(m.Verdict, &v); err == nil {
			verdict = &v
		}
	}
	failureReason := ""
	if m.FailureReason != nil {
		failureReason = *m.FailureReason
	}
	return &domainmedia.Media{
		ID:            m.ID,
		CampaignID:    m.CampaignID,
		ClientID:      m.ClientID,
		Filename:      m.Filename,
		ContentType:   m.ContentType,
		MediaType:     domainmedia.MediaType(m.MediaType),
		ObjectKey:     m.ObjectKey,
		SizeBytes:     m.SizeBytes,
		Status:        domainmedia.Status(m.Status),
		Verdict:       verdict,
		FailureReason: failureReason,
		AnalyzedAt:    m.AnalyzedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
}
