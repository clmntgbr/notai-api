package write

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go-api/internal/domain/activity"
	"go-api/internal/infrastructure/persistence/dbtype"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type activityEventWriteRepository struct {
	db *gorm.DB
}

func NewActivityEventWriteRepository(db *gorm.DB) activity.WriteRepository {
	return &activityEventWriteRepository{db: db}
}

func (r *activityEventWriteRepository) Insert(ctx context.Context, event *activity.Event) (bool, error) {
	if event == nil {
		return false, errors.New("activity event is required")
	}

	payload := event.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	model := ActivityEventModel{
		ID:          event.ID,
		ClientID:    event.ClientID,
		Type:        event.Type,
		ActorType:   event.ActorType,
		ActorUserID: event.ActorUserID,
		ActorName:   event.ActorName,
		Message:     event.Message,
		Payload:     dbtype.JSONB(raw),
		OccurredAt:  event.OccurredAt.UTC(),
		CreatedAt:   createdAt.UTC(),
	}

	result := DBWithContext(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
