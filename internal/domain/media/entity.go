package media

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type Status string

const (
	StatusPendingUpload Status = "pending_upload"
	StatusUploaded      Status = "uploaded"
	StatusProcessing    Status = "processing"
	StatusAnalyzed      Status = "analyzed"
	StatusFailed        Status = "failed"
)

var (
	ErrInvalidTransition      = errors.New("invalid media status transition")
	ErrInvalidFilename        = errors.New("invalid media filename")
	ErrUnsupportedContentType = errors.New("unsupported media file type")
	ErrTooManyFiles           = errors.New("too many files")
	ErrEmptyFileList          = errors.New("files are required")
	ErrInvalidMediaType       = errors.New("invalid media type")
)

const (
	MaxPresignBatch  = 20
	MaxMediaBytes    = 200 << 20 // 200 MiB (videos)
	MaxImageBytes    = 20 << 20  // 20 MiB
	ThumbnailMaxWidth = 400
)

var imageExtensions = map[string]struct{}{
	"jpg": {}, "jpeg": {}, "png": {}, "webp": {},
}

var videoExtensions = map[string]struct{}{
	"mp4": {}, "webm": {}, "mov": {},
}

// Verdict is the campaign-facing aggregate outcome across child contents.
type Verdict struct {
	Label        domaincontent.Label `json:"label"`
	FlaggedCount int                 `json:"flaggedCount"`
	TotalCount   int                 `json:"totalCount"`
	FailedCount  int                 `json:"failedCount"`
}

type Media struct {
	ID          uuid.UUID
	CampaignID  uuid.UUID
	ClientID    uuid.UUID
	Filename    string
	ContentType string
	MediaType   MediaType
	ObjectKey   string
	SizeBytes   *int64
	Status      Status
	Verdict     *Verdict
	AnalyzedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	events []event.DomainEvent
}

func FileExtension(filename string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
}

func DetectMediaType(filename, contentType string) (MediaType, error) {
	ext := FileExtension(filename)
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(ct, "image/"), extIn(ext, imageExtensions):
		if _, ok := imageExtensions[ext]; !ok && ext != "" {
			return "", ErrUnsupportedContentType
		}
		return MediaTypeImage, nil
	case strings.HasPrefix(ct, "video/"), extIn(ext, videoExtensions):
		if _, ok := videoExtensions[ext]; !ok && ext != "" {
			return "", ErrUnsupportedContentType
		}
		return MediaTypeVideo, nil
	default:
		return "", ErrUnsupportedContentType
	}
}

func ValidateFilename(filename string) error {
	ext := FileExtension(filename)
	if ext == "" {
		return ErrUnsupportedContentType
	}
	if _, ok := imageExtensions[ext]; ok {
		return nil
	}
	if _, ok := videoExtensions[ext]; ok {
		return nil
	}
	return ErrUnsupportedContentType
}

func extIn(ext string, set map[string]struct{}) bool {
	_, ok := set[ext]
	return ok
}

func IsMediaObjectKey(key string) bool {
	return strings.Contains(key, "/media/") &&
		!strings.Contains(key, "/frames/") &&
		!IsThumbnailObjectKey(key)
}

func IsFrameObjectKey(key string) bool {
	return strings.Contains(key, "/media/") && strings.Contains(key, "/frames/")
}

func IsThumbnailObjectKey(key string) bool {
	return strings.Contains(key, "/thumbnails/")
}

func NewObjectKey(clientID, campaignID, mediaID uuid.UUID, filename string) string {
	ext := FileExtension(filename)
	key := mediaID.String()
	if ext != "" {
		key += "." + ext
	}
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/media/%s",
		clientID.String(),
		campaignID.String(),
		key,
	)
}

func NewFrameObjectKey(clientID, campaignID, mediaID uuid.UUID, frameIndex int) string {
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/media/%s/frames/%03d.jpg",
		clientID.String(),
		campaignID.String(),
		mediaID.String(),
		frameIndex,
	)
}

func NewThumbnailKey(clientID, campaignID, mediaID uuid.UUID) string {
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/media/thumbnails/%s.jpg",
		clientID.String(),
		campaignID.String(),
		mediaID.String(),
	)
}

func NewPendingUpload(
	campaignID, clientID uuid.UUID,
	filename, contentType string,
) (*Media, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." || filename == ".." {
		return nil, ErrInvalidFilename
	}
	if err := ValidateFilename(filename); err != nil {
		return nil, err
	}
	mediaType, err := DetectMediaType(filename, contentType)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := uuid.New()
	m := &Media{
		ID:          id,
		CampaignID:  campaignID,
		ClientID:    clientID,
		Filename:    filename,
		ContentType: strings.TrimSpace(contentType),
		MediaType:   mediaType,
		ObjectKey:   NewObjectKey(clientID, campaignID, id, filename),
		Status:      StatusPendingUpload,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.recordEvent(MediaUploadRequested{
		ID:          uuid.New().String(),
		MediaID:     m.ID.String(),
		CampaignID:  m.CampaignID.String(),
		ClientID:    m.ClientID.String(),
		Filename:    m.Filename,
		ContentType: m.ContentType,
		MediaType:   string(m.MediaType),
		ObjectKey:   m.ObjectKey,
		Status:      string(m.Status),
		Timestamp:   now,
	})
	return m, nil
}

func (m *Media) PullEvents() []event.DomainEvent {
	events := m.events
	m.events = nil
	return events
}

func (m *Media) recordEvent(e event.DomainEvent) {
	m.events = append(m.events, e)
}

func (m *Media) MarkUploaded(sizeBytes int64, contentType string) error {
	if m.Status != StatusPendingUpload {
		return ErrInvalidTransition
	}
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	now := time.Now().UTC()
	m.SizeBytes = &sizeBytes
	if ct := strings.TrimSpace(contentType); ct != "" {
		m.ContentType = ct
	}
	m.Status = StatusUploaded
	m.UpdatedAt = now
	m.recordStatusChanged(now)
	m.recordEvent(MediaUploaded{
		ID:         uuid.New().String(),
		MediaID:    m.ID.String(),
		CampaignID: m.CampaignID.String(),
		ClientID:   m.ClientID.String(),
		MediaType:  string(m.MediaType),
		ObjectKey:  m.ObjectKey,
		SizeBytes:  sizeBytes,
		Timestamp:  now,
	})
	return nil
}

func (m *Media) StartProcessing() error {
	if m.Status != StatusUploaded && m.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	if m.Status == StatusProcessing {
		return nil
	}
	now := time.Now().UTC()
	m.Status = StatusProcessing
	m.UpdatedAt = now
	m.recordStatusChanged(now)
	m.recordEvent(MediaProcessingStarted{
		ID:         uuid.New().String(),
		MediaID:    m.ID.String(),
		CampaignID: m.CampaignID.String(),
		ClientID:   m.ClientID.String(),
		Timestamp:  now,
	})
	return nil
}

func (m *Media) RenderGlobalVerdict(v Verdict) error {
	if m.Status != StatusProcessing && m.Status != StatusUploaded {
		return ErrInvalidTransition
	}
	if err := domaincontent.ValidateLabel(v.Label); err != nil {
		return err
	}
	now := time.Now().UTC()
	verdict := v
	m.Verdict = &verdict
	m.AnalyzedAt = &now
	m.Status = StatusAnalyzed
	m.UpdatedAt = now
	m.recordStatusChanged(now)
	m.recordEvent(MediaVerdictRendered{
		ID:           uuid.New().String(),
		MediaID:      m.ID.String(),
		CampaignID:   m.CampaignID.String(),
		ClientID:     m.ClientID.String(),
		Label:        string(v.Label),
		FlaggedCount: v.FlaggedCount,
		TotalCount:   v.TotalCount,
		FailedCount:  v.FailedCount,
		Timestamp:    now,
	})
	return nil
}

func (m *Media) MarkFailed(reason string) error {
	if m.Status != StatusPendingUpload &&
		m.Status != StatusUploaded &&
		m.Status != StatusProcessing {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	m.Status = StatusFailed
	m.UpdatedAt = now
	m.recordStatusChanged(now)
	_ = reason
	return nil
}

func (m *Media) recordStatusChanged(at time.Time) {
	label := ""
	if m.Verdict != nil {
		label = string(m.Verdict.Label)
	}
	m.recordEvent(MediaStatusChanged{
		ID:         uuid.New().String(),
		MediaID:    m.ID.String(),
		CampaignID: m.CampaignID.String(),
		ClientID:   m.ClientID.String(),
		Status:     string(m.Status),
		Label:      label,
		Timestamp:  at,
	})
}

// MaxBytesFor returns the upload size limit for a media type.
func MaxBytesFor(t MediaType) int64 {
	if t == MediaTypeVideo {
		return MaxMediaBytes
	}
	return MaxImageBytes
}
