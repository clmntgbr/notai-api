package write

import (
	"time"

	domaincampaign "go-api/internal/domain/campaign"

	"github.com/google/uuid"
)

type CampaignModel struct {
	ID        uuid.UUID  `gorm:"column:id;primaryKey"`
	ClientID  uuid.UUID  `gorm:"column:client_id"`
	Name      string     `gorm:"column:name"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
	StartAt *time.Time `gorm:"column:start_at"`
	EndAt   *time.Time `gorm:"column:end_at"`

	BackgroundStatus       string  `gorm:"column:background_status"`
	BackgroundPendingKey   *string `gorm:"column:background_pending_key"`
	BackgroundThumbnailKey *string `gorm:"column:background_thumbnail_key"`
	BackgroundFilename     *string `gorm:"column:background_filename"`
	BackgroundContentType  *string `gorm:"column:background_content_type"`
	IsDefault              bool    `gorm:"column:is_default"`
}

func (CampaignModel) TableName() string {
	return "campaigns"
}

func optionalNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func campaignModelFromDomain(c *domaincampaign.Campaign) *CampaignModel {
	status := c.BackgroundStatus
	if status == "" {
		status = domaincampaign.BackgroundStatusNone
	}
	return &CampaignModel{
		ID:                     c.ID,
		ClientID:               c.ClientID,
		Name:                   c.Name,
		CreatedAt:              c.CreatedAt,
		UpdatedAt:              c.UpdatedAt,
		DeletedAt:              c.DeletedAt,
		StartAt:              c.StartAt,
		EndAt:                c.EndAt,
		BackgroundStatus:       status,
		BackgroundPendingKey:   optionalNonEmpty(c.BackgroundPendingKey),
		BackgroundThumbnailKey: optionalNonEmpty(c.BackgroundThumbnailKey),
		BackgroundFilename:     optionalNonEmpty(c.BackgroundFilename),
		BackgroundContentType:  optionalNonEmpty(c.BackgroundContentType),
		IsDefault:              c.IsDefault,
	}
}

func campaignDomainFromModel(m *CampaignModel) *domaincampaign.Campaign {
	status := m.BackgroundStatus
	if status == "" {
		status = domaincampaign.BackgroundStatusNone
	}
	return &domaincampaign.Campaign{
		ID:                     m.ID,
		ClientID:               m.ClientID,
		Name:                   m.Name,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
		DeletedAt:              m.DeletedAt,
		StartAt:              m.StartAt,
		EndAt:                m.EndAt,
		BackgroundStatus:       status,
		BackgroundPendingKey:   derefString(m.BackgroundPendingKey),
		BackgroundThumbnailKey: derefString(m.BackgroundThumbnailKey),
		BackgroundFilename:     derefString(m.BackgroundFilename),
		BackgroundContentType:  derefString(m.BackgroundContentType),
		IsDefault:              m.IsDefault,
	}
}
