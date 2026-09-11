package content

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPendingUpload Status = "pending_upload"
	StatusUploaded      Status = "uploaded"
	StatusAnalyzing     Status = "analyzing"
	StatusFailed        Status = "failed"
)

type Label string

const (
	LabelHuman       Label = "human"
	LabelAIGenerated Label = "ai_generated"
	LabelUncertain   Label = "uncertain"
)

var (
	ErrInvalidTransition      = errors.New("invalid content status transition")
	ErrInvalidFilename        = errors.New("invalid content filename")
	ErrUnsupportedContentType = errors.New("unsupported content file type")
	ErrTooManyFiles           = errors.New("too many files")
	ErrEmptyFileList          = errors.New("files are required")
	ErrInvalidLabel           = errors.New("invalid content label")
)

const (
	MaxPresignBatch   = 20
	MaxContentBytes   = 20 << 20 // 20 MiB
	ThumbnailMaxWidth = 400
)

var imageExtensions = map[string]struct{}{
	"jpg":  {},
	"jpeg": {},
	"png":  {},
	"webp": {},
}

func FileExtension(filename string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
}

func ValidateContentFilename(filename string) error {
	ext := FileExtension(filename)
	if ext == "" {
		return ErrUnsupportedContentType
	}
	if _, ok := imageExtensions[ext]; !ok {
		return ErrUnsupportedContentType
	}
	return nil
}

func ValidateLabel(label Label) error {
	switch label {
	case LabelHuman, LabelAIGenerated, LabelUncertain:
		return nil
	default:
		return ErrInvalidLabel
	}
}

func IsContentObjectKey(key string) bool {
	return strings.Contains(key, "/contents/") && !IsThumbnailObjectKey(key)
}

func IsThumbnailObjectKey(key string) bool {
	return strings.Contains(key, "/thumbnails/")
}

func NewThumbnailKey(clientID, campaignID, contentID uuid.UUID) string {
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/contents/thumbnails/%s.jpg",
		clientID.String(),
		campaignID.String(),
		contentID.String(),
	)
}

type Content struct {
	ID           uuid.UUID
	CampaignID   uuid.UUID
	ClientID     uuid.UUID
	Filename     string
	ContentType  string
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Label        *Label

	Status Status

	CreatedAt time.Time
	UpdatedAt time.Time

	events []event.DomainEvent
}

func NewPendingUpload(
	campaignID, clientID uuid.UUID,
	filename, contentType string,
) (*Content, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." || filename == ".." {
		return nil, ErrInvalidFilename
	}
	if err := ValidateContentFilename(filename); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := uuid.New()
	c := &Content{
		ID:          id,
		CampaignID:  campaignID,
		ClientID:    clientID,
		Filename:    filename,
		ContentType: strings.TrimSpace(contentType),
		ObjectKey:   NewObjectKey(clientID, campaignID, id, filename),
		Status:      StatusPendingUpload,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	c.recordEvent(ContentCreated{
		ID:          uuid.New().String(),
		ContentID:   c.ID.String(),
		CampaignID:  c.CampaignID.String(),
		ClientID:    c.ClientID.String(),
		Filename:    c.Filename,
		ContentType: c.ContentType,
		ObjectKey:   c.ObjectKey,
		Status:      string(c.Status),
		Timestamp:   now,
	})
	return c, nil
}

func NewObjectKey(clientID, campaignID, contentID uuid.UUID, filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	key := contentID.String()
	if ext != "" {
		key += "." + ext
	}
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/contents/%s",
		clientID.String(),
		campaignID.String(),
		key,
	)
}

func (c *Content) PullEvents() []event.DomainEvent {
	events := c.events
	c.events = nil
	return events
}

func (c *Content) recordEvent(e event.DomainEvent) {
	c.events = append(c.events, e)
}

func (c *Content) MarkUploaded(sizeBytes int64, contentType, thumbnailKey string) error {
	if c.Status != StatusPendingUpload {
		return ErrInvalidTransition
	}
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	now := time.Now().UTC()
	c.SizeBytes = &sizeBytes
	if ct := strings.TrimSpace(contentType); ct != "" {
		c.ContentType = ct
	}
	thumb := strings.TrimSpace(thumbnailKey)
	if thumb == "" {
		return errors.New("thumbnail key is required")
	}
	c.ThumbnailKey = &thumb
	c.Status = StatusUploaded
	c.UpdatedAt = now
	c.recordStatusChanged(now)
	c.recordEvent(ContentUploaded{
		ID:           uuid.New().String(),
		ContentID:    c.ID.String(),
		CampaignID:   c.CampaignID.String(),
		ClientID:     c.ClientID.String(),
		ObjectKey:    c.ObjectKey,
		ThumbnailKey: thumb,
		SizeBytes:    sizeBytes,
		Timestamp:    now,
	})
	return nil
}

func (c *Content) StartAnalyzing() error {
	if c.Status != StatusUploaded {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	c.Label = nil
	c.Status = StatusAnalyzing
	c.UpdatedAt = now
	c.recordStatusChanged(now)
	return nil
}

func (c *Content) CompleteAnalysis(label Label) error {
	if c.Status != StatusAnalyzing {
		return ErrInvalidTransition
	}
	if err := ValidateLabel(label); err != nil {
		return err
	}
	now := time.Now().UTC()
	c.Label = &label
	c.Status = StatusUploaded
	c.UpdatedAt = now
	c.recordStatusChanged(now)
	return nil
}

func (c *Content) MarkFailed() error {
	if c.Status != StatusPendingUpload &&
		c.Status != StatusUploaded &&
		c.Status != StatusAnalyzing {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	c.Status = StatusFailed
	c.UpdatedAt = now
	c.recordStatusChanged(now)
	return nil
}

func (c *Content) SetThumbnailKey(key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		c.ThumbnailKey = nil
		return
	}
	c.ThumbnailKey = &key
	c.UpdatedAt = time.Now().UTC()
}

func (c *Content) recordStatusChanged(at time.Time) {
	label := ""
	if c.Label != nil {
		label = string(*c.Label)
	}
	c.recordEvent(ContentStatusChanged{
		ID:         uuid.New().String(),
		ContentID:  c.ID.String(),
		CampaignID: c.CampaignID.String(),
		ClientID:   c.ClientID.String(),
		Status:     string(c.Status),
		Label:      label,
		Timestamp:  at,
	})
}
