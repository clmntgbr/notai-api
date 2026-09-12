package write

import (
	"time"

	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
)

type ActivityEventModel struct {
	ID          uuid.UUID    `gorm:"column:id;primaryKey"`
	ClientID    uuid.UUID    `gorm:"column:client_id"`
	Type        string       `gorm:"column:type"`
	ActorType   string       `gorm:"column:actor_type"`
	ActorUserID *uuid.UUID   `gorm:"column:actor_user_id"`
	ActorName   string       `gorm:"column:actor_name"`
	Message     string       `gorm:"column:message"`
	Payload     dbtype.JSONB `gorm:"column:payload"`
	OccurredAt  time.Time    `gorm:"column:occurred_at"`
	CreatedAt   time.Time    `gorm:"column:created_at"`
}

func (ActivityEventModel) TableName() string {
	return "activity_events"
}
