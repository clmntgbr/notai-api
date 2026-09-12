package read

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"
	"go-api/internal/infrastructure/persistence/dbtype"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mediaRow struct {
	ID          uuid.UUID
	CampaignID  uuid.UUID
	ClientID    uuid.UUID
	Filename    string
	ContentType string
	MediaType   string
	ObjectKey   string
	SizeBytes   *int64
	Status      string
	Verdict     dbtype.JSONB
	AnalyzedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (mediaRow) TableName() string { return "media" }

type mediaReadRepository struct {
	db *gorm.DB
}

func NewMediaReadRepository(db *gorm.DB) domainmedia.MediaReadRepository {
	return &mediaReadRepository{db: db}
}

const mediaSelectCols = "id, campaign_id, client_id, filename, content_type, media_type, object_key, " +
	"size_bytes, status, verdict, analyzed_at, created_at, updated_at"

func (r *mediaReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainmedia.MediaView, error) {
	var row mediaRow
	err := r.db.WithContext(ctx).
		Select(mediaSelectCols).
		First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toMediaView(row), nil
}

func (r *mediaReadRepository) FindPageByCampaignID(
	ctx context.Context,
	clientID, campaignID uuid.UUID,
	query paginate.PaginateQuery,
) ([]domainmedia.MediaView, int64, error) {
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
		Model(&mediaRow{}).
		Where("client_id = ? AND campaign_id = ?", clientID, campaignID)

	if query.Search != "" {
		db = db.Where("filename ILIKE ?", "%"+query.Search+"%")
	}

	db, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []mediaRow
	if err := db.Select(mediaSelectCols).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]domainmedia.MediaView, 0, len(rows))
	for _, row := range rows {
		views = append(views, *toMediaView(row))
	}
	return views, total, nil
}

func (r *mediaReadRepository) FindContentsByMediaID(
	ctx context.Context,
	mediaID uuid.UUID,
) ([]domainmedia.ContentChildView, error) {
	var rows []struct {
		ID           uuid.UUID
		MediaID      uuid.UUID
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
	err := r.db.WithContext(ctx).
		Table("contents").
		Where("media_id = ?", mediaID).
		Order("frame_index ASC NULLS FIRST, created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domainmedia.ContentChildView, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainmedia.ContentChildView{
			ID:           row.ID,
			MediaID:      row.MediaID,
			FrameIndex:   row.FrameIndex,
			TimestampMs:  row.TimestampMs,
			ObjectKey:    row.ObjectKey,
			ThumbnailKey: row.ThumbnailKey,
			SizeBytes:    row.SizeBytes,
			Status:       row.Status,
			Label:        row.Label,
			Confidence:   row.Confidence,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *mediaReadRepository) CountStatsByClientID(
	ctx context.Context,
	clientID uuid.UUID,
) (*domainmedia.MediaStats, error) {
	var row struct {
		PendingUpload int64
		Uploaded      int64
		Processing    int64
		Analyzed      int64
		Failed        int64
		Human         int64
		AIGenerated   int64
		Uncertain     int64
	}
	err := r.db.WithContext(ctx).
		Table("media").
		Select(`
			COUNT(*) FILTER (WHERE status = 'pending_upload') AS pending_upload,
			COUNT(*) FILTER (WHERE status = 'uploaded') AS uploaded,
			COUNT(*) FILTER (WHERE status = 'processing') AS processing,
			COUNT(*) FILTER (WHERE status = 'analyzed') AS analyzed,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'human') AS human,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'uncertain') AS uncertain
		`).
		Where("client_id = ?", clientID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}

	monthly, err := r.countMonthlyControlsByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	return &domainmedia.MediaStats{
		PendingUpload:   row.PendingUpload,
		Uploaded:        row.Uploaded,
		Processing:      row.Processing,
		Analyzed:        row.Analyzed,
		Failed:          row.Failed,
		Human:           row.Human,
		AIGenerated:     row.AIGenerated,
		Uncertain:       row.Uncertain,
		MonthlyControls: monthly,
	}, nil
}

func (r *mediaReadRepository) countMonthlyControlsByClientID(
	ctx context.Context,
	clientID uuid.UUID,
) ([]domainmedia.MediaMonthlyStats, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -5, 0)

	var rows []struct {
		Month         string
		PendingUpload int64
		Uploaded      int64
		Processing    int64
		Analyzed      int64
		Failed        int64
		Human         int64
		AIGenerated   int64
		Uncertain     int64
	}
	err := r.db.WithContext(ctx).
		Table("media").
		Select(`
			to_char(date_trunc('month', created_at AT TIME ZONE 'UTC'), 'YYYY-MM') AS month,
			COUNT(*) FILTER (WHERE status = 'pending_upload') AS pending_upload,
			COUNT(*) FILTER (WHERE status = 'uploaded') AS uploaded,
			COUNT(*) FILTER (WHERE status = 'processing') AS processing,
			COUNT(*) FILTER (WHERE status = 'analyzed') AS analyzed,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'human') AS human,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'ai_generated') AS ai_generated,
			COUNT(*) FILTER (WHERE verdict->>'label' = 'uncertain') AS uncertain
		`).
		Where("client_id = ? AND created_at >= ?", clientID, start).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byMonth := make(map[string]domainmedia.MediaMonthlyStats, len(rows))
	for _, row := range rows {
		byMonth[row.Month] = domainmedia.MediaMonthlyStats{
			Month:         row.Month,
			PendingUpload: row.PendingUpload,
			Uploaded:      row.Uploaded,
			Processing:    row.Processing,
			Analyzed:      row.Analyzed,
			Failed:        row.Failed,
			Human:         row.Human,
			AIGenerated:   row.AIGenerated,
			Uncertain:     row.Uncertain,
		}
	}

	out := make([]domainmedia.MediaMonthlyStats, 0, 6)
	for i := 0; i < 6; i++ {
		monthStart := start.AddDate(0, i, 0)
		key := monthStart.Format("2006-01")
		if stats, ok := byMonth[key]; ok {
			out = append(out, stats)
			continue
		}
		out = append(out, domainmedia.MediaMonthlyStats{Month: key})
	}
	return out, nil
}

func toMediaView(row mediaRow) *domainmedia.MediaView {
	var verdict *domainmedia.Verdict
	if len(row.Verdict) > 0 {
		var v domainmedia.Verdict
		if err := json.Unmarshal(row.Verdict, &v); err == nil {
			verdict = &v
		}
	}
	return &domainmedia.MediaView{
		ID:          row.ID,
		CampaignID:  row.CampaignID,
		ClientID:    row.ClientID,
		Filename:    row.Filename,
		ContentType: row.ContentType,
		MediaType:   domainmedia.MediaType(row.MediaType),
		ObjectKey:   row.ObjectKey,
		SizeBytes:   row.SizeBytes,
		Status:      domainmedia.Status(row.Status),
		Verdict:     verdict,
		AnalyzedAt:  row.AnalyzedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
