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
	StartAt   *time.Time
	EndAt     *time.Time

	BackgroundStatus       string
	BackgroundPendingKey   *string
	BackgroundThumbnailKey *string
	BackgroundFilename     *string
	BackgroundContentType  *string
	IsDefault              bool
}

func (campaignRow) TableName() string { return "campaigns" }

type campaignContentCountRow struct {
	CampaignID  uuid.UUID
	Failed      int64
	Human       int64
	AIGenerated int64
	Uncertain   int64
}

type campaignReadRepository struct {
	db *gorm.DB
}

func NewCampaignReadRepository(db *gorm.DB) domaincampaign.CampaignReadRepository {
	return &campaignReadRepository{db: db}
}

const campaignSelectCols = "id, client_id, name, created_at, updated_at, start_at, end_at, is_default, " +
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
	view := toCampaignView(row)
	if err := r.attachContentCounts(ctx, []*domaincampaign.CampaignView{view}); err != nil {
		return nil, err
	}
	return view, nil
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
	view := toCampaignView(row)
	if err := r.attachContentCounts(ctx, []*domaincampaign.CampaignView{view}); err != nil {
		return nil, err
	}
	return view, nil
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
	case "start_at":
		query.SortBy = "start_at"
	case "end_at":
		query.SortBy = "end_at"
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
	ptrs := make([]*domaincampaign.CampaignView, 0, len(rows))
	for _, row := range rows {
		view := toCampaignView(row)
		views = append(views, *view)
		ptrs = append(ptrs, &views[len(views)-1])
	}
	if err := r.attachContentCounts(ctx, ptrs); err != nil {
		return nil, 0, err
	}
	return views, total, nil
}

func (r *campaignReadRepository) attachContentCounts(
	ctx context.Context,
	views []*domaincampaign.CampaignView,
) error {
	if len(views) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(views))
	indexByID := make(map[uuid.UUID]*domaincampaign.CampaignView, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
		indexByID[view.ID] = view
	}

	var rows []campaignContentCountRow
	err := r.db.WithContext(ctx).
		Table("contents").
		Select(`
			campaign_id,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE label = 'human') AS human,
			COUNT(*) FILTER (WHERE label = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE label = 'uncertain') AS uncertain
		`).
		Where("campaign_id IN ?", ids).
		Group("campaign_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}

	for _, row := range rows {
		view, ok := indexByID[row.CampaignID]
		if !ok {
			continue
		}
		view.ContentFailedCount = row.Failed
		view.ContentHumanCount = row.Human
		view.ContentAIGeneratedCount = row.AIGenerated
		view.ContentUncertainCount = row.Uncertain
	}
	return nil
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
		StartAt:                row.StartAt,
		EndAt:                  row.EndAt,
		BackgroundStatus:       status,
		BackgroundPendingKey:   derefString(row.BackgroundPendingKey),
		BackgroundThumbnailKey: derefString(row.BackgroundThumbnailKey),
		BackgroundFilename:     derefString(row.BackgroundFilename),
		BackgroundContentType:  derefString(row.BackgroundContentType),
	}
}
