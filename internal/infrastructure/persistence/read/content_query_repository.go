package read

import (
	"context"
	"time"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contentReadRepository struct {
	db *gorm.DB
}

func NewContentReadRepository(db *gorm.DB) domaincontent.ContentReadRepository {
	return &contentReadRepository{db: db}
}

func (r *contentReadRepository) CountByClientIDAndStatus(
	ctx context.Context,
	clientID uuid.UUID,
	status string,
) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Where("media.client_id = ? AND contents.status = ?", clientID, status).
		Count(&count).Error
	return count, err
}

func (r *contentReadRepository) CountByClientIDAndStatusInPeriod(
	ctx context.Context,
	clientID uuid.UUID,
	status string,
	from, to time.Time,
) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Where(
			"media.client_id = ? AND contents.status = ? AND media.analyzed_at >= ? AND media.analyzed_at < ?",
			clientID,
			status,
			from,
			to,
		).
		Count(&count).Error
	return count, err
}

// CountQuotaUnitsByClientIDInPeriod counts contents that consume monthly verification quota:
// currently analyzing, plus analyzed in the billing period.
// Analyzed units use media.analyzed_at when the media is finalized; otherwise contents.updated_at
// (set when the content verdict is rendered) so in-flight finals still consume a slot.
// Uploaded / pending_upload are not reserved — quota is checked when analysis starts.
func (r *contentReadRepository) CountQuotaUnitsByClientIDInPeriod(
	ctx context.Context,
	clientID uuid.UUID,
	from, to time.Time,
) (int64, error) {
	var contentCount int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM contents
		JOIN media ON media.id = contents.media_id
		WHERE media.client_id = @clientID
		  AND (
			contents.status = 'analyzing'
			OR (
				contents.status = 'analyzed'
				AND COALESCE(media.analyzed_at, contents.updated_at) >= @from
				AND COALESCE(media.analyzed_at, contents.updated_at) < @to
			)
		  )
	`, map[string]any{
		"clientID": clientID,
		"from":     from,
		"to":       to,
	}).Scan(&contentCount).Error
	return contentCount, err
}

func (r *contentReadRepository) CountByWorkspaceIDAndStatus(
	ctx context.Context,
	workspaceID uuid.UUID,
	status string,
) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contents").
		Joins("JOIN media ON media.id = contents.media_id").
		Joins("JOIN clients ON clients.id = media.client_id").
		Where("clients.workspace_id = ? AND contents.status = ?", workspaceID, status).
		Count(&count).Error
	return count, err
}

func (r *contentReadRepository) CountQuotaUnitsByWorkspaceIDInPeriod(
	ctx context.Context,
	workspaceID uuid.UUID,
	from, to time.Time,
) (int64, error) {
	var contentCount int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM contents
		JOIN media ON media.id = contents.media_id
		JOIN clients ON clients.id = media.client_id
		WHERE clients.workspace_id = @workspaceID
		  AND (
			contents.status = 'analyzing'
			OR (
				contents.status = 'analyzed'
				AND COALESCE(media.analyzed_at, contents.updated_at) >= @from
				AND COALESCE(media.analyzed_at, contents.updated_at) < @to
			)
		  )
	`, map[string]any{
		"workspaceID": workspaceID,
		"from":        from,
		"to":          to,
	}).Scan(&contentCount).Error
	return contentCount, err
}
