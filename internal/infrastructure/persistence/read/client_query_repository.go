package read

import (
	"context"
	"errors"
	"time"

	"go-api/internal/domain/paginate"
	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clientRow struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (clientRow) TableName() string { return "clients" }

type userClientRow struct {
	UserID   uuid.UUID
	ClientID uuid.UUID
}

func (userClientRow) TableName() string { return "user_clients" }

type clientReadRepository struct {
	db *gorm.DB
}

func NewClientReadRepository(db *gorm.DB) domainclient.ClientReadRepository {
	return &clientReadRepository{db: db}
}

func (r *clientReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainclient.ClientView, error) {
	var row clientRow
	err := r.db.WithContext(ctx).
		Select("id", "name", "created_at", "updated_at").
		First(&row, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	memberIDs, err := r.loadMemberIDs(ctx, id)
	if err != nil {
		return nil, err
	}

	return toClientView(row, memberIDs), nil
}

func (r *clientReadRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]domainclient.ClientView, error) {
	var memberships []userClientRow
	if err := r.db.WithContext(ctx).
		Select("user_id", "client_id").
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []domainclient.ClientView{}, nil
	}

	clientIDs := make([]uuid.UUID, 0, len(memberships))
	for _, m := range memberships {
		clientIDs = append(clientIDs, m.ClientID)
	}

	var rows []clientRow
	if err := r.db.WithContext(ctx).
		Select("id", "name", "created_at", "updated_at").
		Where("id IN ?", clientIDs).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	views := make([]domainclient.ClientView, 0, len(rows))
	for _, row := range rows {
		memberIDs, err := r.loadMemberIDs(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		views = append(views, *toClientView(row, memberIDs))
	}
	return views, nil
}

func (r *clientReadRepository) FindPageByUserID(
	ctx context.Context,
	userID uuid.UUID,
	query paginate.PaginateQuery,
) ([]domainclient.ClientView, int64, error) {
	switch query.SortBy {
	case "", "created_at":
		query.SortBy = "clients.created_at"
	case "updated_at":
		query.SortBy = "clients.updated_at"
	case "name":
		query.SortBy = "clients.name"
	default:
		query.SortBy = "clients.created_at"
	}

	db := r.db.WithContext(ctx).
		Model(&clientRow{}).
		Joins("INNER JOIN user_clients ON user_clients.client_id = clients.id").
		Where("user_clients.user_id = ?", userID)

	if query.Search != "" {
		db = db.Where("clients.name ILIKE ?", "%"+query.Search+"%")
	}

	db, total, err := Paginate(db, query)
	if err != nil {
		return nil, 0, err
	}

	var rows []clientRow
	if err := db.Select("clients.id", "clients.name", "clients.created_at", "clients.updated_at").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	views := make([]domainclient.ClientView, 0, len(rows))
	for _, row := range rows {
		memberIDs, err := r.loadMemberIDs(ctx, row.ID)
		if err != nil {
			return nil, 0, err
		}
		views = append(views, *toClientView(row, memberIDs))
	}
	return views, total, nil
}

func (r *clientReadRepository) loadMemberIDs(ctx context.Context, clientID uuid.UUID) ([]uuid.UUID, error) {
	var memberships []userClientRow
	if err := r.db.WithContext(ctx).
		Select("user_id", "client_id").
		Where("client_id = ?", clientID).
		Find(&memberships).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(memberships))
	for _, m := range memberships {
		ids = append(ids, m.UserID)
	}
	return ids, nil
}

func toClientView(row clientRow, memberIDs []uuid.UUID) *domainclient.ClientView {
	return &domainclient.ClientView{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		MemberIDs: memberIDs,
	}
}
