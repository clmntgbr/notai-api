package media

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"
)

type memStorage struct {
	deleted        []string
	deletedThumbs  []string
	failOnKey      string
	failOnThumbKey string
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
	if key == s.failOnKey {
		return errors.New("delete failed")
	}
	s.deleted = append(s.deleted, key)
	return nil
}
func (s *memStorage) DeleteThumbnail(_ context.Context, key string) error {
	if key == s.failOnThumbKey {
		return errors.New("delete thumb failed")
	}
	s.deletedThumbs = append(s.deletedThumbs, key)
	return nil
}
func (s *memStorage) PresignedPutURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func TestDeleteObjectsOnDeletedHandler_DeletesDedupedKeys(t *testing.T) {
	storage := &memStorage{}
	h := NewDeleteObjectsOnDeletedHandler(storage)

	evt := domainmedia.MediaDeleted{
		ID:           "evt-1",
		MediaID:      "media-1",
		ObjectKey:    "media/original.jpg",
		ThumbnailKey: "media/thumbs/original.jpg",
		Contents: []domainmedia.MediaDeletedContentKey{
			{ObjectKey: "media/original.jpg", ThumbnailKey: "media/thumbs/content.jpg"},
			{ObjectKey: "media/frames/001.jpg", ThumbnailKey: "media/thumbs/frame001.jpg"},
		},
		Timestamp: time.Now().UTC(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := h.Handle(context.Background(), payload); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if len(storage.deleted) != 2 {
		t.Fatalf("object deletes: got %v", storage.deleted)
	}
	if len(storage.deletedThumbs) != 3 {
		t.Fatalf("thumb deletes: got %v", storage.deletedThumbs)
	}
}

func TestDeleteObjectsOnDeletedHandler_RetryableOnStorageError(t *testing.T) {
	storage := &memStorage{failOnKey: "bad-key"}
	h := NewDeleteObjectsOnDeletedHandler(storage)

	payload, _ := json.Marshal(domainmedia.MediaDeleted{
		ID:        "evt-1",
		MediaID:   "media-1",
		ObjectKey: "bad-key",
	})
	err := h.Handle(context.Background(), payload)
	var retryable *messaging.RetryableError
	if !errors.As(err, &retryable) {
		t.Fatalf("want retryable, got %v", err)
	}
}
