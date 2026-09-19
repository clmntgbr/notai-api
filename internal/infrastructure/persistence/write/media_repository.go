package write

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mediaWriteRepository struct {
	db *gorm.DB
}

func NewMediaWriteRepository(db *gorm.DB) domainmedia.MediaWriteRepository {
	return &mediaWriteRepository{db: db}
}

func (r *mediaWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *mediaWriteRepository) Save(ctx context.Context, media *domainmedia.Media) error {
	return DBWithContext(ctx, r.db).Create(mediaModelFromDomain(media)).Error
}

func (r *mediaWriteRepository) Update(ctx context.Context, media *domainmedia.Media) error {
	return DBWithContext(ctx, r.db).Save(mediaModelFromDomain(media)).Error
}

func (r *mediaWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainmedia.Media, error) {
	var model MediaModel
	err := DBWithContext(ctx, r.db).
		Where("deleted_at IS NULL").
		First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) GetByObjectKey(
	ctx context.Context,
	objectKey string,
) (*domainmedia.Media, error) {
	var model MediaModel
	err := DBWithContext(ctx, r.db).
		Where("deleted_at IS NULL").
		First(&model, "object_key = ?", objectKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaDomainFromModel(&model), nil
}

func (r *mediaWriteRepository) ListProcessingUpdatedBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) ([]*domainmedia.Media, error) {
	if limit <= 0 {
		limit = 100
	}
	var models []MediaModel
	err := DBWithContext(ctx, r.db).
		Where("deleted_at IS NULL").
		Where("status = ? AND updated_at < ?", string(domainmedia.StatusProcessing), before.UTC()).
		Order("updated_at ASC").
		Limit(limit).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domainmedia.Media, 0, len(models))
	for i := range models {
		out = append(out, mediaDomainFromModel(&models[i]))
	}
	return out, nil
}

func (r *mediaWriteRepository) ListActiveByCampaignID(
	ctx context.Context,
	campaignID uuid.UUID,
) ([]*domainmedia.Media, error) {
	var models []MediaModel
	err := DBWithContext(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("campaign_id = ? AND deleted_at IS NULL", campaignID).
		Order("created_at ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	out := make([]*domainmedia.Media, 0, len(models))
	for i := range models {
		out = append(out, mediaDomainFromModel(&models[i]))
	}
	return out, nil
}
