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

// CountQuotaUnitsByClientIDInPeriod counts contents that already consume monthly verification quota:
// in-flight (uploaded/analyzing) plus analyzed in the billing period. Also includes pending_upload
// media that do not yet have a content row.
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
			contents.status IN ('uploaded', 'analyzing')
			OR (
				contents.status = 'analyzed'
				AND media.analyzed_at >= @from
				AND media.analyzed_at < @to
			)
		  )
	`, map[string]any{
		"clientID": clientID,
		"from":     from,
		"to":       to,
	}).Scan(&contentCount).Error
	if err != nil {
		return 0, err
	}

	var pendingMedia int64
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM media
		WHERE client_id = @clientID
		  AND status = 'pending_upload'
		  AND NOT EXISTS (
			SELECT 1 FROM contents c WHERE c.media_id = media.id
		  )
	`, map[string]any{"clientID": clientID}).Scan(&pendingMedia).Error
	if err != nil {
		return 0, err
	}
	return contentCount + pendingMedia, nil
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
			contents.status IN ('uploaded', 'analyzing')
			OR (
				contents.status = 'analyzed'
				AND media.analyzed_at >= @from
				AND media.analyzed_at < @to
			)
		  )
	`, map[string]any{
		"workspaceID": workspaceID,
		"from":        from,
		"to":          to,
	}).Scan(&contentCount).Error
	if err != nil {
		return 0, err
	}

	var pendingMedia int64
	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM media
		JOIN clients ON clients.id = media.client_id
		WHERE clients.workspace_id = @workspaceID
		  AND media.status = 'pending_upload'
		  AND NOT EXISTS (
			SELECT 1 FROM contents c WHERE c.media_id = media.id
		  )
	`, map[string]any{"workspaceID": workspaceID}).Scan(&pendingMedia).Error
	if err != nil {
		return 0, err
	}
	return contentCount + pendingMedia, nil
}
