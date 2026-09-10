package write

import (
	"time"

	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
)

type ClientModel struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ClientModel) TableName() string {
	return "clients"
}

type UserClientModel struct {
	UserID    uuid.UUID `gorm:"column:user_id;primaryKey"`
	ClientID  uuid.UUID `gorm:"column:client_id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (UserClientModel) TableName() string {
	return "user_clients"
}

func clientModelFromDomain(c *domainclient.Client) *ClientModel {
	return &ClientModel{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func clientDomainFromModel(m *ClientModel, memberIDs []uuid.UUID) *domainclient.Client {
	return &domainclient.Client{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		MemberIDs: memberIDs,
	}
}
