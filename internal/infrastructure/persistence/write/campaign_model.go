package write

import (
	"time"

	domaincampaign "go-api/internal/domain/campaign"

	"github.com/google/uuid"
)

type CampaignModel struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	ClientID  uuid.UUID `gorm:"column:client_id"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (CampaignModel) TableName() string {
	return "campaigns"
}

func campaignModelFromDomain(c *domaincampaign.Campaign) *CampaignModel {
	return &CampaignModel{
		ID:        c.ID,
		ClientID:  c.ClientID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func campaignDomainFromModel(m *CampaignModel) *domaincampaign.Campaign {
	return &domaincampaign.Campaign{
		ID:        m.ID,
		ClientID:  m.ClientID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
