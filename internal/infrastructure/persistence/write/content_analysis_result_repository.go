package write

import (
	"context"
	"errors"

	"go-api/internal/domain/analysisresult"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contentAnalysisResultRepository struct {
	db *gorm.DB
}

func NewContentAnalysisResultRepository(db *gorm.DB) analysisresult.WriteRepository {
	return &contentAnalysisResultRepository{db: db}
}

func (r *contentAnalysisResultRepository) Upsert(ctx context.Context, result *analysisresult.Result) error {
	if result == nil {
		return errors.New("result is required")
	}

	signalsJSON, err := result.SignalsJSON()
	if err != nil {
		return err
	}

	var errMsg *string
	if result.Error != "" {
		msg := result.Error
		errMsg = &msg
	}

	model := ContentAnalysisResultModel{
		ID:             result.ID,
		ContentID:      result.ContentID,
		DetectorName:   result.DetectorName,
		Status:         string(result.Status),
		Signals:        dbtype.JSONB(signalsJSON),
		Error:          errMsg,
		StartedAt:      result.StartedAt,
		CompletedAt:    result.CompletedAt,
		RulesetVersion: result.RulesetVersion,
	}

	return DBWithContext(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "content_id"}, {Name: "detector_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status",
			"signals",
			"error",
			"started_at",
			"completed_at",
			"ruleset_version",
		}),
	}).Create(&model).Error
}

func (r *contentAnalysisResultRepository) FindByContentID(
	ctx context.Context,
	contentID uuid.UUID,
) ([]analysisresult.Result, error) {
	var rows []ContentAnalysisResultModel
	if err := DBWithContext(ctx, r.db).
		Where("content_id = ?", contentID).
		Order("detector_name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]analysisresult.Result, 0, len(rows))
	for _, row := range rows {
		signals, err := analysisresult.SignalsFromJSON(row.Signals)
		if err != nil {
			return nil, err
		}
		errText := ""
		if row.Error != nil {
			errText = *row.Error
		}
		out = append(out, analysisresult.Result{
			ID:             row.ID,
			ContentID:      row.ContentID,
			DetectorName:   row.DetectorName,
			Status:         analysisresult.Status(row.Status),
			Signals:        signals,
			Error:          errText,
			StartedAt:      row.StartedAt,
			CompletedAt:    row.CompletedAt,
			Weight:         1,
			RulesetVersion: row.RulesetVersion,
		})
	}
	return out, nil
}
