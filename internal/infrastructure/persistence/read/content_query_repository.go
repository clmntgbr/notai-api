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
	MediaID      uuid.UUID
	CampaignID   uuid.UUID
	ClientID     uuid.UUID
	Filename     string
	ContentType  string
	MediaType    string
	FrameIndex   *int
	TimestampMs  *int64
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Status       string
	Label        *string
	Confidence   *float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type contentReadRepository struct {
	db *gorm.DB
}

func NewContentReadRepository(db *gorm.DB) domaincontent.ContentReadRepository {
	return &contentReadRepository{db: db}
}

const contentSelectCols = `
	contents.id,
	contents.media_id,
	media.campaign_id,
	media.client_id,
	media.filename,
	media.content_type,
	media.media_type,
	contents.frame_index,
	contents.timestamp_ms,
	contents.object_key,
	contents.thumbnail_key,
	contents.size_bytes,
	contents.status,
	contents.label,
	contents.confidence,
	contents.created_at,
	contents.updated_at
`

func (r *contentReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domaincontent.ContentView, error) {
	var row contentRow
	err := r.db.WithContext(ctx).
		Table("contents").
		Select(contentSelectCols).
		Joins("JOIN media ON media.id = contents.media_id").
		Where("contents.id = ?", id).
		First(&row).Error
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
		query.SortBy = "contents.created_at"
	case "updated_at":
		query.SortBy = "contents.updated_at"
	case "filename":
		query.SortBy = "media.filename"
	case "status":
		query.SortBy = "contents.status"
	default:
		query.SortBy = "contents.created_at"
	}

	db := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Where("media.client_id = ?", clientID)
	if campaignID != uuid.Nil {
		db = db.Where("media.campaign_id = ?", campaignID)
	}

	if query.Search != "" {
		db = db.Where("media.filename ILIKE ?", "%"+query.Search+"%")
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
		Analyzed      int64
		Failed        int64
		Human         int64
		AIGenerated   int64
		Uncertain     int64
	}
	err := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Select(`
			COUNT(*) FILTER (WHERE contents.status = 'pending_upload') AS pending_upload,
			COUNT(*) FILTER (WHERE contents.status = 'uploaded') AS uploaded,
			COUNT(*) FILTER (WHERE contents.status = 'analyzing') AS analyzing,
			COUNT(*) FILTER (WHERE contents.status = 'analyzed') AS analyzed,
			COUNT(*) FILTER (WHERE contents.status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE contents.label = 'human') AS human,
			COUNT(*) FILTER (WHERE contents.label = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE contents.label = 'uncertain') AS uncertain
		`).
		Where("media.client_id = ?", clientID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}

	monthly, err := r.countMonthlyControlsByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	return &domaincontent.ContentStats{
		PendingUpload:   row.PendingUpload,
		Uploaded:        row.Uploaded,
		Analyzing:       row.Analyzing,
		Analyzed:        row.Analyzed,
		Failed:          row.Failed,
		Human:           row.Human,
		AIGenerated:     row.AIGenerated,
		Uncertain:       row.Uncertain,
		MonthlyControls: monthly,
	}, nil
}

func (r *contentReadRepository) countMonthlyControlsByClientID(
	ctx context.Context,
	clientID uuid.UUID,
) ([]domaincontent.ContentMonthlyStats, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -5, 0)

	var rows []struct {
		Month         string
		PendingUpload int64
		Uploaded      int64
		Analyzing     int64
		Analyzed      int64
		Failed        int64
		Human         int64
		AIGenerated   int64
		Uncertain     int64
	}
	err := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Select(`
			to_char(date_trunc('month', contents.created_at AT TIME ZONE 'UTC'), 'YYYY-MM') AS month,
			COUNT(*) FILTER (WHERE contents.status = 'pending_upload') AS pending_upload,
			COUNT(*) FILTER (WHERE contents.status = 'uploaded') AS uploaded,
			COUNT(*) FILTER (WHERE contents.status = 'analyzing') AS analyzing,
			COUNT(*) FILTER (WHERE contents.status = 'analyzed') AS analyzed,
			COUNT(*) FILTER (WHERE contents.status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE contents.label = 'human') AS human,
			COUNT(*) FILTER (WHERE contents.label = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE contents.label = 'uncertain') AS uncertain
		`).
		Where("media.client_id = ? AND contents.created_at >= ?", clientID, start).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byMonth := make(map[string]domaincontent.ContentMonthlyStats, len(rows))
	for _, row := range rows {
		byMonth[row.Month] = domaincontent.ContentMonthlyStats{
			Month:         row.Month,
			PendingUpload: row.PendingUpload,
			Uploaded:      row.Uploaded,
			Analyzing:     row.Analyzing,
			Analyzed:      row.Analyzed,
			Failed:        row.Failed,
			Human:         row.Human,
			AIGenerated:   row.AIGenerated,
			Uncertain:     row.Uncertain,
		}
	}

	out := make([]domaincontent.ContentMonthlyStats, 0, 6)
	for i := 0; i < 6; i++ {
		monthStart := start.AddDate(0, i, 0)
		key := monthStart.Format("2006-01")
		if stats, ok := byMonth[key]; ok {
			out = append(out, stats)
			continue
		}
		out = append(out, domaincontent.ContentMonthlyStats{Month: key})
	}
	return out, nil
}

func toContentView(row contentRow) *domaincontent.ContentView {
	var label *domaincontent.Label
	if row.Label != nil && *row.Label != "" {
		l := domaincontent.Label(*row.Label)
		label = &l
	}
	return &domaincontent.ContentView{
		ID:           row.ID,
		MediaID:      row.MediaID,
		CampaignID:   row.CampaignID,
		ClientID:     row.ClientID,
		Filename:     row.Filename,
		ContentType:  row.ContentType,
		MediaType:    row.MediaType,
		FrameIndex:   row.FrameIndex,
		TimestampMs:  row.TimestampMs,
		ObjectKey:    row.ObjectKey,
		ThumbnailKey: row.ThumbnailKey,
		SizeBytes:    row.SizeBytes,
		Status:       domaincontent.Status(row.Status),
		Label:        label,
		Confidence:   row.Confidence,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
