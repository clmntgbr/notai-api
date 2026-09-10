package write

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type campaignWriteRepository struct {
	db *gorm.DB
}

func NewCampaignWriteRepository(db *gorm.DB) domaincampaign.CampaignWriteRepository {
	return &campaignWriteRepository{db: db}
}

func (r *campaignWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *campaignWriteRepository) Save(ctx context.Context, campaign *domaincampaign.Campaign) error {
	return DBWithContext(ctx, r.db).Create(campaignModelFromDomain(campaign)).Error
}

func (r *campaignWriteRepository) Update(ctx context.Context, campaign *domaincampaign.Campaign) error {
	return DBWithContext(ctx, r.db).Save(campaignModelFromDomain(campaign)).Error
}

func (r *campaignWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domaincampaign.Campaign, error) {
	var model CampaignModel
	err := DBWithContext(ctx, r.db).
		Where("deleted_at IS NULL").
		First(&model, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return campaignDomainFromModel(&model), nil
}

func (r *campaignWriteRepository) GetByBackgroundPendingKey(
	ctx context.Context,
	pendingKey string,
) (*domaincampaign.Campaign, error) {
	var model CampaignModel
	err := DBWithContext(ctx, r.db).
		Where("deleted_at IS NULL").
		First(&model, "background_pending_key = ?", pendingKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return campaignDomainFromModel(&model), nil
}
