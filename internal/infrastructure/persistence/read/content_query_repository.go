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
	Label        *string
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
	"thumbnail_key, size_bytes, status, label, created_at, updated_at"

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

func (r *contentReadRepository) FindPageByClientID(
	ctx context.Context,
	clientID uuid.UUID,
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
		Where("client_id = ?", clientID)
	if campaignID != uuid.Nil {
		db = db.Where("campaign_id = ?", campaignID)
	}

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

func (r *contentReadRepository) CountStatsByClientID(
	ctx context.Context,
	clientID uuid.UUID,
) (*domaincontent.ContentStats, error) {
	var row struct {
		PendingUpload int64
		Uploaded      int64
		Analyzing     int64
		Failed        int64
		Human         int64
		AIGenerated   int64
		Uncertain     int64
	}
	err := r.db.WithContext(ctx).
		Table("contents").
		Select(`
			COUNT(*) FILTER (WHERE status = 'pending_upload') AS pending_upload,
			COUNT(*) FILTER (WHERE status = 'uploaded') AS uploaded,
			COUNT(*) FILTER (WHERE status = 'analyzing') AS analyzing,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE label = 'human') AS human,
			COUNT(*) FILTER (WHERE label = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE label = 'uncertain') AS uncertain
		`).
		Where("client_id = ?", clientID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &domaincontent.ContentStats{
		PendingUpload: row.PendingUpload,
		Uploaded:      row.Uploaded,
		Analyzing:     row.Analyzing,
		Failed:        row.Failed,
		Human:         row.Human,
		AIGenerated:   row.AIGenerated,
		Uncertain:     row.Uncertain,
	}, nil
}

func toContentView(row contentRow) *domaincontent.ContentView {
	var label *domaincontent.Label
	if row.Label != nil && *row.Label != "" {
		l := domaincontent.Label(*row.Label)
		label = &l
	}
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
		Label:        label,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
