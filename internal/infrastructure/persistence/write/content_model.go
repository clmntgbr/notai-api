package write

import (
	"time"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

type ContentModel struct {
	ID           uuid.UUID `gorm:"column:id;primaryKey"`
	CampaignID   uuid.UUID `gorm:"column:campaign_id"`
	ClientID     uuid.UUID `gorm:"column:client_id"`
	Filename     string    `gorm:"column:filename"`
	ContentType  string    `gorm:"column:content_type"`
	ObjectKey    string    `gorm:"column:object_key"`
	ThumbnailKey *string   `gorm:"column:thumbnail_key"`
	SizeBytes    *int64    `gorm:"column:size_bytes"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ContentModel) TableName() string {
	return "contents"
}

func contentModelFromDomain(c *domaincontent.Content) *ContentModel {
	return &ContentModel{
		ID:           c.ID,
		CampaignID:   c.CampaignID,
		ClientID:     c.ClientID,
		Filename:     c.Filename,
		ContentType:  c.ContentType,
		ObjectKey:    c.ObjectKey,
		ThumbnailKey: c.ThumbnailKey,
		SizeBytes:    c.SizeBytes,
		Status:       string(c.Status),
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func contentDomainFromModel(m *ContentModel) *domaincontent.Content {
	return &domaincontent.Content{
		ID:           m.ID,
		CampaignID:   m.CampaignID,
		ClientID:     m.ClientID,
		Filename:     m.Filename,
		ContentType:  m.ContentType,
		ObjectKey:    m.ObjectKey,
		ThumbnailKey: m.ThumbnailKey,
		SizeBytes:    m.SizeBytes,
		Status:       domaincontent.Status(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
