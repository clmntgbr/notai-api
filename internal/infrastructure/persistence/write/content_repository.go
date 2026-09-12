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
	var row contentWithMediaRow
	err := DBWithContext(ctx, r.db).
		Table("contents").
		Select("contents.*, media.campaign_id, media.client_id").
		Joins("JOIN media ON media.id = contents.media_id").
		Where("contents.id = ?", id).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return contentDomainFromModel(&row.ContentModel, row.CampaignID, row.ClientID), nil
}

func (r *contentWriteRepository) GetByObjectKey(
	ctx context.Context,
	objectKey string,
) (*domaincontent.Content, error) {
	var row contentWithMediaRow
	err := DBWithContext(ctx, r.db).
		Table("contents").
		Select("contents.*, media.campaign_id, media.client_id").
		Joins("JOIN media ON media.id = contents.media_id").
		Where("contents.object_key = ?", objectKey).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return contentDomainFromModel(&row.ContentModel, row.CampaignID, row.ClientID), nil
}

func (r *contentWriteRepository) ListByMediaID(
	ctx context.Context,
	mediaID uuid.UUID,
) ([]domaincontent.Content, error) {
	var rows []contentWithMediaRow
	err := DBWithContext(ctx, r.db).
		Table("contents").
		Select("contents.*, media.campaign_id, media.client_id").
		Joins("JOIN media ON media.id = contents.media_id").
		Where("contents.media_id = ?", mediaID).
		Order("contents.frame_index ASC NULLS FIRST, contents.created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domaincontent.Content, 0, len(rows))
	for i := range rows {
		out = append(out, *contentDomainFromModel(&rows[i].ContentModel, rows[i].CampaignID, rows[i].ClientID))
	}
	return out, nil
}
