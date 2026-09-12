package write

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
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
	err := DBWithContext(ctx, r.db).First(&model, "object_key = ?", objectKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mediaDomainFromModel(&model), nil
}
