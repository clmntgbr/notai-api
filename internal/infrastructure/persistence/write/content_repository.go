package write

import (
	"context"
	"errors"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contentWriteRepository struct {
	db *gorm.DB
}

func NewContentWriteRepository(db *gorm.DB) domaincontent.ContentWriteRepository {
	return &contentWriteRepository{db: db}
}

func (r *contentWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *contentWriteRepository) Save(ctx context.Context, content *domaincontent.Content) error {
	return DBWithContext(ctx, r.db).Create(contentModelFromDomain(content)).Error
}

func (r *contentWriteRepository) Update(ctx context.Context, content *domaincontent.Content) error {
	return DBWithContext(ctx, r.db).Save(contentModelFromDomain(content)).Error
}

func (r *contentWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domaincontent.Content, error) {
	var model ContentModel
	err := DBWithContext(ctx, r.db).First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return contentDomainFromModel(&model), nil
}

func (r *contentWriteRepository) GetByObjectKey(
	ctx context.Context,
	objectKey string,
) (*domaincontent.Content, error) {
	var model ContentModel
	err := DBWithContext(ctx, r.db).First(&model, "object_key = ?", objectKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return contentDomainFromModel(&model), nil
}
