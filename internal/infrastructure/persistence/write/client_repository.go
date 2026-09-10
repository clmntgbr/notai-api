package write

import (
	"context"
	"errors"
	"time"

	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type clientWriteRepository struct {
	db *gorm.DB
}

func NewClientWriteRepository(db *gorm.DB) domainclient.ClientWriteRepository {
	return &clientWriteRepository{db: db}
}

func (r *clientWriteRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ContextWithTx(ctx, tx))
	})
}

func (r *clientWriteRepository) Save(ctx context.Context, client *domainclient.Client) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Create(clientModelFromDomain(client)).Error; err != nil {
		return err
	}
	return r.ReplaceMembers(ctx, client.ID, client.MemberIDs)
}

func (r *clientWriteRepository) Update(ctx context.Context, client *domainclient.Client) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Save(clientModelFromDomain(client)).Error; err != nil {
		return err
	}
	return r.ReplaceMembers(ctx, client.ID, client.MemberIDs)
}

func (r *clientWriteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Where("client_id = ?", id).Delete(&UserClientModel{}).Error; err != nil {
		return err
	}
	return db.Delete(&ClientModel{}, id).Error
}

func (r *clientWriteRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainclient.Client, error) {
	db := DBWithContext(ctx, r.db)
	var model ClientModel
	err := db.First(&model, "id = ?", id).Error
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
	return clientDomainFromModel(&model, memberIDs), nil
}

func (r *clientWriteRepository) ReplaceMembers(ctx context.Context, clientID uuid.UUID, memberIDs []uuid.UUID) error {
	db := DBWithContext(ctx, r.db)
	if err := db.Where("client_id = ?", clientID).Delete(&UserClientModel{}).Error; err != nil {
		return err
	}
	if len(memberIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	rows := make([]UserClientModel, 0, len(memberIDs))
	for _, userID := range memberIDs {
		rows = append(rows, UserClientModel{
			UserID:    userID,
			ClientID:  clientID,
			CreatedAt: now,
		})
	}
	return db.Create(&rows).Error
}

func (r *clientWriteRepository) loadMemberIDs(ctx context.Context, clientID uuid.UUID) ([]uuid.UUID, error) {
	var rows []UserClientModel
	err := DBWithContext(ctx, r.db).
		Where("client_id = ?", clientID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	return ids, nil
}
