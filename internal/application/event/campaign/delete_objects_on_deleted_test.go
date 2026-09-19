package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"go-api/internal/application/messaging"
	domaincampaign "go-api/internal/domain/campaign"
)

type memStorage struct {
	deleted       []string
	deletedThumbs []string
	failPending   bool
	failThumb     bool
}

func (s *memStorage) Get(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (s *memStorage) Put(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (s *memStorage) GetThumbnail(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *memStorage) PutThumbnail(context.Context, string, io.Reader, int64, string) error {
	return nil
}
func (s *memStorage) Delete(_ context.Context, key string) error {
	if s.failPending {
		return errors.New("delete pending failed")
	}
	s.deleted = append(s.deleted, key)
	return nil
}
func (s *memStorage) DeleteThumbnail(_ context.Context, key string) error {
	if s.failThumb {
		return errors.New("delete thumb failed")
	}
	s.deletedThumbs = append(s.deletedThumbs, key)
	return nil
}
func (s *memStorage) PresignedPutURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func TestDeleteObjectsOnDeletedHandler_DeletesPendingAndThumbnail(t *testing.T) {
	storage := &memStorage{}
	h := NewDeleteObjectsOnDeletedHandler(storage)

	payload, err := json.Marshal(domaincampaign.CampaignDeleted{
		ID:                     "evt-1",
		CampaignID:             "campaign-1",
		BackgroundPendingKey:   "campaigns/pending.jpg",
		BackgroundThumbnailKey: "campaigns/thumb.jpg",
		Timestamp:              time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := h.Handle(context.Background(), payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(storage.deleted) != 1 || storage.deleted[0] != "campaigns/pending.jpg" {
		t.Fatalf("pending deletes: %v", storage.deleted)
	}
	if len(storage.deletedThumbs) != 1 || storage.deletedThumbs[0] != "campaigns/thumb.jpg" {
		t.Fatalf("thumb deletes: %v", storage.deletedThumbs)
	}
}

func TestDeleteObjectsOnDeletedHandler_NoKeysIsNoop(t *testing.T) {
	storage := &memStorage{}
	h := NewDeleteObjectsOnDeletedHandler(storage)

	payload, _ := json.Marshal(domaincampaign.CampaignDeleted{
		ID:         "evt-1",
		CampaignID: "campaign-1",
	})
	if err := h.Handle(context.Background(), payload); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(storage.deleted) != 0 || len(storage.deletedThumbs) != 0 {
		t.Fatalf("unexpected deletes: %v %v", storage.deleted, storage.deletedThumbs)
	}
}

func TestDeleteObjectsOnDeletedHandler_RetryableOnStorageError(t *testing.T) {
	storage := &memStorage{failThumb: true}
	h := NewDeleteObjectsOnDeletedHandler(storage)

	payload, _ := json.Marshal(domaincampaign.CampaignDeleted{
		ID:                     "evt-1",
		CampaignID:             "campaign-1",
		BackgroundThumbnailKey: "campaigns/thumb.jpg",
	})
	err := h.Handle(context.Background(), payload)
	var retryable *messaging.RetryableError
	if !errors.As(err, &retryable) {
		t.Fatalf("want retryable, got %v", err)
	}
}
