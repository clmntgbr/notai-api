package write

import (
	"time"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type ContentModel struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey"`
	MediaID      uuid.UUID `gorm:"column:media_id"`
	FrameIndex   *int      `gorm:"column:frame_index"`
	TimestampMs  *int64    `gorm:"column:timestamp_ms"`
	ObjectKey    string    `gorm:"column:object_key"`
	ThumbnailKey *string   `gorm:"column:thumbnail_key"`
	SizeBytes    *int64    `gorm:"column:size_bytes"`
	Status       string    `gorm:"column:status"`
	Label        *string   `gorm:"column:label"`
	Confidence   *float64  `gorm:"column:confidence"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ContentModel) TableName() string {
	return "contents"
}

func contentModelFromDomain(c *domaincontent.Content) *ContentModel {
	return &ContentModel{
		ID:           c.ID,
		MediaID:      c.MediaID,
		FrameIndex:   c.FrameIndex,
		TimestampMs:  c.TimestampMs,
		ObjectKey:    c.ObjectKey,
		ThumbnailKey: c.ThumbnailKey,
		SizeBytes:    c.SizeBytes,
		Status:       string(c.Status),
		Label:        labelStringPtr(c.Label),
		Confidence:   c.Confidence,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

type contentWithMediaRow struct {
	ContentModel
	CampaignID  uuid.UUID `gorm:"column:campaign_id"`
	ClientID    uuid.UUID `gorm:"column:client_id"`
}

func contentDomainFromModel(m *ContentModel, campaignID, clientID uuid.UUID) *domaincontent.Content {
	return &domaincontent.Content{
		ID:           m.ID,
		MediaID:      m.MediaID,
		FrameIndex:   m.FrameIndex,
		TimestampMs:  m.TimestampMs,
		ObjectKey:    m.ObjectKey,
		ThumbnailKey: m.ThumbnailKey,
		SizeBytes:    m.SizeBytes,
		Status:       domaincontent.Status(m.Status),
		Label:        labelFromStringPtr(m.Label),
		Confidence:   m.Confidence,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		CampaignID:   campaignID,
		ClientID:     clientID,
	}
}

func labelStringPtr(label *domaincontent.Label) *string {
	if label == nil {
		return nil
	}
	s := string(*label)
	return &s
}

func labelFromStringPtr(s *string) *domaincontent.Label {
	if s == nil || *s == "" {
		return nil
	}
	label := domaincontent.Label(*s)
	return &label
}
