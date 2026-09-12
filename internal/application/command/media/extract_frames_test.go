package media

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/event"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type memMediaRepo struct {
	byID  map[uuid.UUID]*domainmedia.Media
	saved []*domainmedia.Media
}

func (r *memMediaRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memMediaRepo) Save(ctx context.Context, media *domainmedia.Media) error {
	r.byID[media.ID] = cloneMedia(media)
	return nil
}
func (r *memMediaRepo) Update(ctx context.Context, media *domainmedia.Media) error {
	r.byID[media.ID] = cloneMedia(media)
	r.saved = append(r.saved, cloneMedia(media))
	return nil
}
func (r *memMediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*domainmedia.Media, error) {
	m := r.byID[id]
	if m == nil {
		return nil, nil
	}
	return cloneMedia(m), nil
}
func (r *memMediaRepo) GetByObjectKey(ctx context.Context, objectKey string) (*domainmedia.Media, error) {
	for _, m := range r.byID {
		if m.ObjectKey == objectKey {
			return cloneMedia(m), nil
		}
	}
	return nil, nil
}

func cloneMedia(m *domainmedia.Media) *domainmedia.Media {
	cp := *m
	return &cp
}

type memContentRepo struct {
	items []domaincontent.Content
}

func (r *memContentRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memContentRepo) Save(ctx context.Context, content *domaincontent.Content) error {
	r.items = append(r.items, *content)
	return nil
}
func (r *memContentRepo) Update(ctx context.Context, content *domaincontent.Content) error {
	return nil
}
func (r *memContentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domaincontent.Content, error) {
	return nil, nil
}
func (r *memContentRepo) GetByObjectKey(ctx context.Context, objectKey string) (*domaincontent.Content, error) {
	return nil, nil
}
func (r *memContentRepo) ListByMediaID(ctx context.Context, mediaID uuid.UUID) ([]domaincontent.Content, error) {
	out := make([]domaincontent.Content, 0)
	for _, c := range r.items {
		if c.MediaID == mediaID {
			out = append(out, c)
		}
	}
	return out, nil
}

type memOutbox struct{}

func (memOutbox) StoreEvents(ctx context.Context, events []event.DomainEvent) error {
	return nil
}
func (memOutbox) FetchUnpublished(ctx context.Context, limit int) ([]port.OutboxMessage, error) {
	return nil, nil
}
func (memOutbox) MarkPublished(ctx context.Context, ids []uuid.UUID) error { return nil }

type memStorage struct {
	objects map[string][]byte
}

func (s *memStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	b, ok := s.objects[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (s *memStorage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	b, _ := io.ReadAll(r)
	s.objects[key] = b
	return nil
}
func (s *memStorage) GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.Get(ctx, key)
}
func (s *memStorage) PutThumbnail(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	return s.Put(ctx, key, r, size, contentType)
}
func (s *memStorage) Delete(ctx context.Context, key string) error { return nil }
func (s *memStorage) DeleteThumbnail(ctx context.Context, key string) error {
	return nil
}
func (s *memStorage) PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", nil
}

type stubExtractor struct {
	frames []port.ExtractedFrame
	err    error
}

func (s stubExtractor) Extract(ctx context.Context, videoPath string) ([]port.ExtractedFrame, error) {
	return s.frames, s.err
}

func TestExtractFramesHandler_SuccessCreatesContents(t *testing.T) {
	mediaID := uuid.New()
	m := &domainmedia.Media{
		ID:         mediaID,
		CampaignID: uuid.New(),
		ClientID:   uuid.New(),
		MediaType:  domainmedia.MediaTypeVideo,
		ObjectKey:  "clients/a/campaigns/b/media/" + mediaID.String() + ".mp4",
		Status:     domainmedia.StatusProcessing,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	mediaRepo := &memMediaRepo{byID: map[uuid.UUID]*domainmedia.Media{mediaID: m}}
	contentRepo := &memContentRepo{}
	storage := &memStorage{objects: map[string][]byte{m.ObjectKey: []byte("video")}}
	frames := make([]port.ExtractedFrame, 10)
	for i := range frames {
		frames[i] = port.ExtractedFrame{
			Index: i, TimestampMs: int64(i * 1000), JPEGBytes: []byte("jpeg"),
		}
	}
	extractor := stubExtractor{frames: frames}

	h := NewExtractFramesHandler(mediaRepo, contentRepo, memOutbox{}, storage, extractor)
	if err := h.Handle(context.Background(), ExtractFramesCommand{MediaID: mediaID}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(contentRepo.items) != 10 {
		t.Fatalf("expected 10 contents, got %d", len(contentRepo.items))
	}
	if contentRepo.items[0].FrameIndex == nil || *contentRepo.items[0].FrameIndex != 0 {
		t.Fatalf("frame index: %+v", contentRepo.items[0].FrameIndex)
	}
}

func TestExtractFramesHandler_ExtractorFailureMarksMediaFailed(t *testing.T) {
	mediaID := uuid.New()
	m := &domainmedia.Media{
		ID:         mediaID,
		CampaignID: uuid.New(),
		ClientID:   uuid.New(),
		MediaType:  domainmedia.MediaTypeVideo,
		ObjectKey:  "clients/a/campaigns/b/media/" + mediaID.String() + ".mp4",
		Status:     domainmedia.StatusProcessing,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	mediaRepo := &memMediaRepo{byID: map[uuid.UUID]*domainmedia.Media{mediaID: m}}
	contentRepo := &memContentRepo{}
	storage := &memStorage{objects: map[string][]byte{m.ObjectKey: []byte("video")}}
	extractor := stubExtractor{err: errors.New("ffmpeg boom")}

	h := NewExtractFramesHandler(mediaRepo, contentRepo, memOutbox{}, storage, extractor)
	err := h.Handle(context.Background(), ExtractFramesCommand{MediaID: mediaID})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(contentRepo.items) != 0 {
		t.Fatalf("expected no contents, got %d", len(contentRepo.items))
	}
	fresh, _ := mediaRepo.GetByID(context.Background(), mediaID)
	if fresh.Status != domainmedia.StatusFailed {
		t.Fatalf("expected failed media, got %s", fresh.Status)
	}
}
