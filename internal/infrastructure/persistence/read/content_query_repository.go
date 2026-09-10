package read

import (
	"context"
	"errors"
	"time"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contentRow struct {
	ID           uuid.UUID
	CampaignID   uuid.UUID
	ClientID     uuid.UUID
	Filename     string
	ContentType  string
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Status       string
	AnalyzedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (contentRow) TableName() string { return "contents" }

type contentReadRepository struct {
	db *gorm.DB
}

func NewContentReadRepository(db *gorm.DB) domaincontent.ContentReadRepository {
	return &contentReadRepository{db: db}
}

const contentSelectCols = "id, campaign_id, client_id, filename, content_type, object_key, " +
	"thumbnail_key, size_bytes, status, analyzed_at, created_at, updated_at"

func (r *contentReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domaincontent.ContentView, error) {
	var row contentRow
	err := r.db.WithContext(ctx).
		Select(contentSelectCols).
		First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toContentView(row), nil
}

func (r *contentReadRepository) FindPageByCampaignID(
	ctx context.Context,
	campaignID uuid.UUID,
	query paginate.PaginateQuery,
) ([]domaincontent.ContentView, int64, error) {
	switch query.SortBy {
	case "", "created_at":
		query.SortBy = "created_at"
	case "updated_at":
		query.SortBy = "updated_at"
	case "filename":
		query.SortBy = "filename"
	case "status":
		query.SortBy = "status"
	default:
		query.SortBy = "created_at"
	}

	db := r.db.WithContext(ctx).
		Model(&contentRow{}).
		Where("campaign_id = ?", campaignID)

	if query.Search != "" {
		db = db.Where("filename ILIKE ?", "%"+query.Search+"%")
	}

	db, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []contentRow
	if err := db.Select(contentSelectCols).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]domaincontent.ContentView, 0, len(rows))
	for _, row := range rows {
		views = append(views, *toContentView(row))
	}
	return views, total, nil
}

func toContentView(row contentRow) *domaincontent.ContentView {
	return &domaincontent.ContentView{
		ID:           row.ID,
		CampaignID:   row.CampaignID,
		ClientID:     row.ClientID,
		Filename:     row.Filename,
		ContentType:  row.ContentType,
		ObjectKey:    row.ObjectKey,
		ThumbnailKey: row.ThumbnailKey,
		SizeBytes:    row.SizeBytes,
		Status:       domaincontent.Status(row.Status),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		AnalyzedAt:   row.AnalyzedAt,
	}
}
