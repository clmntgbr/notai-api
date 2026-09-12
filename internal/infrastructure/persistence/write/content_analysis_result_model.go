package write

import (
	"time"

	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
)

type ContentAnalysisResultModel struct {
	ID           uuid.UUID    `gorm:"column:id;primaryKey"`
	ContentID    uuid.UUID    `gorm:"column:content_id"`
	DetectorName string       `gorm:"column:detector_name"`
	Status       string       `gorm:"column:status"`
	Signals      dbtype.JSONB `gorm:"column:signals"`
	Error        *string      `gorm:"column:error"`
	StartedAt    time.Time    `gorm:"column:started_at"`
	CompletedAt  time.Time    `gorm:"column:completed_at"`
}

func (ContentAnalysisResultModel) TableName() string {
	return "content_analysis_results"
}
