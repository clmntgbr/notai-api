package read

import (
	"context"
	"errors"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type campaignRow struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time

	BackgroundStatus       string
	BackgroundPendingKey   *string
	BackgroundThumbnailKey *string
	BackgroundFilename     *string
	BackgroundContentType  *string
	IsDefault              bool
}

func (campaignRow) TableName() string { return "campaigns" }

type campaignReadRepository struct {
	db *gorm.DB
}

func NewCampaignReadRepository(db *gorm.DB) domaincampaign.CampaignReadRepository {
	return &campaignReadRepository{db: db}
}

const campaignSelectCols = "id, client_id, name, created_at, updated_at, is_default, " +
	"background_status, background_pending_key, background_thumbnail_key, " +
	"background_filename, background_content_type"

func (r *campaignReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domaincampaign.CampaignView, error) {
	var row campaignRow
	err := r.db.WithContext(ctx).
		Select(campaignSelectCols).
		Where("deleted_at IS NULL").
		First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toCampaignView(row), nil
}

func (r *campaignReadRepository) FindDefaultByClientID(
	ctx context.Context,
	clientID uuid.UUID,
) (*domaincampaign.CampaignView, error) {
	var row campaignRow
	err := r.db.WithContext(ctx).
		Select(campaignSelectCols).
		Where("deleted_at IS NULL AND client_id = ? AND is_default = true", clientID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toCampaignView(row), nil
}

func (r *campaignReadRepository) FindPageByClientID(
	ctx context.Context,
	clientID uuid.UUID,
	query paginate.PaginateQuery,
) ([]domaincampaign.CampaignView, int64, error) {
	switch query.SortBy {
	case "", "created_at":
		query.SortBy = "created_at"
	case "updated_at":
		query.SortBy = "updated_at"
	case "name":
		query.SortBy = "name"
	default:
		query.SortBy = "created_at"
	}

	db := r.db.WithContext(ctx).
		Model(&campaignRow{}).
		Where("client_id = ?", clientID).
		Where("is_default = false").
		Where("deleted_at IS NULL")

	if query.Search != "" {
		db = db.Where("name ILIKE ?", "%"+query.Search+"%")
	}

	db, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []campaignRow
	if err := db.Select(campaignSelectCols).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]domaincampaign.CampaignView, 0, len(rows))
	for _, row := range rows {
		views = append(views, *toCampaignView(row))
	}
	return views, total, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toCampaignView(row campaignRow) *domaincampaign.CampaignView {
	status := row.BackgroundStatus
	if status == "" {
		status = domaincampaign.BackgroundStatusNone
	}
	return &domaincampaign.CampaignView{
		ID:                     row.ID,
		ClientID:               row.ClientID,
		Name:                   row.Name,
		IsDefault:              row.IsDefault,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		BackgroundStatus:       status,
		BackgroundPendingKey:   derefString(row.BackgroundPendingKey),
		BackgroundThumbnailKey: derefString(row.BackgroundThumbnailKey),
		BackgroundFilename:     derefString(row.BackgroundFilename),
		BackgroundContentType:  derefString(row.BackgroundContentType),
	}
}
