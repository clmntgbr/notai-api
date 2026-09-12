package read

import (
	"context"
	"encoding/json"
	"time"

	"go-api/internal/domain/activity"
	"go-api/internal/domain/paginate"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type activityEventRow struct {
	ID          uuid.UUID
	ClientID    uuid.UUID
	Type        string
	ActorType   string
	ActorUserID *uuid.UUID
	ActorName   string
	Message     string
	Payload     dbtype.JSONB
	OccurredAt  time.Time
}

func (activityEventRow) TableName() string { return "activity_events" }

type activityEventReadRepository struct {
	db *gorm.DB
}

func NewActivityEventReadRepository(db *gorm.DB) activity.ReadRepository {
	return &activityEventReadRepository{db: db}
}

func (r *activityEventReadRepository) FindPageByClientID(
	ctx context.Context,
	clientID uuid.UUID,
	query paginate.PaginateQuery,
) ([]activity.View, int64, error) {
	db := r.db.WithContext(ctx).
		Model(&activityEventRow{}).
		Where("client_id = ?", clientID)

	if query.Search != "" {
		like := "%" + query.Search + "%"
		db = db.Where("message ILIKE ? OR actor_name ILIKE ?", like, like)
	}

	paged, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []activityEventRow
	if err := paged.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]activity.View, 0, len(rows))
	for _, row := range rows {
		payload := map[string]any{}
		if len(row.Payload) > 0 {
			_ = json.Unmarshal(row.Payload, &payload)
		}
		views = append(views, activity.View{
			ID:          row.ID,
			ClientID:    row.ClientID,
			Type:        row.Type,
			ActorType:   row.ActorType,
			ActorUserID: row.ActorUserID,
			ActorName:   row.ActorName,
			Message:     row.Message,
			Payload:     payload,
			OccurredAt:  row.OccurredAt,
		})
	}
	return views, total, nil
}
