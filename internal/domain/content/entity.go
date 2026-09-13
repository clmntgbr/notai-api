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
	StatusAnalyzed      Status = "analyzed"
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

func FileExtension(filename string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
}

func ValidateLabel(label Label) error {
	switch label {
	case LabelHuman, LabelAIGenerated, LabelUncertain:
		return nil
	default:
		return ErrInvalidLabel
	}
}

// IsContentObjectKey reports whether the key is an analyzable content object
// (legacy /contents/ path or extracted /media/.../frames/ frame).
func IsContentObjectKey(key string) bool {
	if IsThumbnailObjectKey(key) {
		return false
	}
	if strings.Contains(key, "/contents/") {
		return true
	}
	return strings.Contains(key, "/media/") && strings.Contains(key, "/frames/")
}

func IsThumbnailObjectKey(key string) bool {
	return strings.Contains(key, "/thumbnails/")
}

func NewThumbnailKey(clientID, campaignID, contentID uuid.UUID) string {
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/media/thumbnails/contents/%s.jpg",
		clientID.String(),
		campaignID.String(),
		contentID.String(),
	)
}

// Content is one analyzable unit (full image or a video frame).
type Content struct {
	ID           uuid.UUID
	MediaID      uuid.UUID
	FrameIndex   *int
	TimestampMs  *int64
	ObjectKey    string
	ThumbnailKey *string
	SizeBytes    *int64
	Label        *Label
	Confidence   *float64
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Ownership context for event payloads (not persisted on contents table).
	CampaignID uuid.UUID
	ClientID   uuid.UUID

	events []event.DomainEvent
}

// NewFromMedia creates an uploaded content unit ready for analysis.
func NewFromMedia(
	mediaID, campaignID, clientID uuid.UUID,
	objectKey string,
	frameIndex *int,
	timestampMs *int64,
	sizeBytes int64,
	thumbnailKey string,
) (*Content, error) {
	if mediaID == uuid.Nil {
		return nil, errors.New("media id is required")
	}
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, errors.New("object key is required")
	}
	now := time.Now().UTC()
	id := uuid.New()
	c := &Content{
		ID:          id,
		MediaID:     mediaID,
		FrameIndex:  frameIndex,
		TimestampMs: timestampMs,
		ObjectKey:   objectKey,
		Status:      StatusUploaded,
		CampaignID:  campaignID,
		ClientID:    clientID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	c.SizeBytes = &sizeBytes
	thumb := strings.TrimSpace(thumbnailKey)
	if thumb != "" {
		c.ThumbnailKey = &thumb
	}

	c.recordEvent(ContentCreated{
		ID:         uuid.New().String(),
		ContentID:  c.ID.String(),
		MediaID:    c.MediaID.String(),
		CampaignID: campaignID.String(),
		ClientID:   clientID.String(),
		ObjectKey:  c.ObjectKey,
		Status:     string(c.Status),
		Timestamp:  now,
	})
	c.recordEvent(ContentUploaded{
		ID:           uuid.New().String(),
		ContentID:    c.ID.String(),
		MediaID:      c.MediaID.String(),
		CampaignID:   campaignID.String(),
		ClientID:     clientID.String(),
		ObjectKey:    c.ObjectKey,
		ThumbnailKey: thumb,
		SizeBytes:    sizeBytes,
		Timestamp:    now,
	})
	return c, nil
}

func (c *Content) PullEvents() []event.DomainEvent {
	events := c.events
	c.events = nil
	return events
}

func (c *Content) recordEvent(e event.DomainEvent) {
	c.events = append(c.events, e)
}

func (c *Content) StartAnalyzing() error {
	if c.Status != StatusUploaded {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	c.Label = nil
	c.Confidence = nil
	c.Status = StatusAnalyzing
	c.UpdatedAt = now
	c.recordStatusChanged(now)
	return nil
}

func (c *Content) StartAnalysis() error {
	return c.StartAnalyzing()
}

func (c *Content) CompleteAnalysis(label Label) error {
	return c.RenderVerdict(Verdict{Label: label, Confidence: 0, Signals: nil})
}

func (c *Content) RenderVerdict(verdict Verdict) error {
	if c.Status != StatusAnalyzing {
		return ErrInvalidTransition
	}
	if err := ValidateLabel(verdict.Label); err != nil {
		return err
	}
	now := time.Now().UTC()
	label := verdict.Label
	c.Label = &label
	confidence := verdict.Confidence
	c.Confidence = &confidence
	c.Status = StatusAnalyzed
	c.UpdatedAt = now
	c.recordEvent(ContentVerdictRendered{
		ID:         uuid.New().String(),
		ContentID:  c.ID.String(),
		MediaID:    c.MediaID.String(),
		CampaignID: c.CampaignID.String(),
		ClientID:   c.ClientID.String(),
		Label:      string(label),
		Confidence: confidence,
		Signals:    verdict.Signals,
		Timestamp:  now,
	})
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
		MediaID:    c.MediaID.String(),
		CampaignID: c.CampaignID.String(),
		ClientID:   c.ClientID.String(),
		Status:     string(c.Status),
		Label:      label,
		Timestamp:  at,
	})
}

// IsTerminal reports whether analysis for this content has finished.
func (c *Content) IsTerminal() bool {
	return c.Status == StatusAnalyzed || c.Status == StatusFailed
}
